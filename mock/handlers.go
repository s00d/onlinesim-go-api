package mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

// Handlers run under m.mu and return the HTTP status and the JSON body.

func ok(fields map[string]any) (int, any) {
	out := map[string]any{"response": "1"}
	for k, v := range fields {
		out[k] = v
	}
	return http.StatusOK, out
}

func fail(code string) (int, any) {
	return http.StatusOK, map[string]any{"response": code}
}

func tzidOf(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.URL.Query().Get("tzid"), 10, 64)
	return id
}

func emptyPage() map[string]any {
	return map[string]any{"current_page": 1, "data": []any{}, "from": nil, "to": nil, "last_page": 1, "per_page": 10, "total": 0}
}

func (m *Server) handleBalance(_ *http.Request) (int, any) {
	if code := m.state.balanceError; code != "" {
		m.state.balanceError = ""
		return fail(code)
	}
	// Like the real API: balance is a decimal string, zbalance and income are numbers.
	return ok(map[string]any{"balance": fmt.Sprintf("%.3f", m.state.Balance), "zbalance": m.state.Zbalance, "income": m.state.Income})
}

func (m *Server) handleProfile(_ *http.Request) (int, any) {
	return ok(map[string]any{"profile": map[string]any{
		"id": 1, "name": "mock", "username": "mock", "email": "mock@example.com",
		"locale": "en", "currency": "USD", "api_access": true, "number_reject": []string{},
		"webhook_url": m.state.WebhookURL,
	}})
}

func (m *Server) handleSetProfile(r *http.Request) (int, any) {
	var body struct {
		Profile map[string]*string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return fail("ERROR_PARAMS")
	}
	hook, present := body.Profile["webhook_url"]
	if !present {
		return ok(nil)
	}
	if hook != nil {
		u, err := url.ParseRequestURI(*hook)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return http.StatusOK, map[string]any{"response": map[string]any{"webhook_url": []string{"The webhook url must be a valid URL."}}}
		}
	}
	m.state.WebhookURL = hook
	return ok(map[string]any{"user": map[string]any{"id": 1}})
}

func (m *Server) handleWebhookLogs(_ *http.Request) (int, any) {
	return ok(map[string]any{"data": emptyPage()})
}

func (m *Server) handlePaymentHistory(_ *http.Request) (int, any) {
	return ok(map[string]any{"orders": emptyPage(), "paylist": []any{}})
}

func (m *Server) handlePrice(r *http.Request) (int, any) {
	service := r.URL.Query().Get("service")
	if service == "" {
		return fail("ERROR_NO_SERVICE")
	}
	price := DefaultSmsScript().Price
	for _, s := range m.state.scripts {
		if s.Service == service {
			price = s.Price
			break
		}
	}
	return ok(map[string]any{"price": price})
}

func (m *Server) handleGetNum(r *http.Request) (int, any) {
	q := r.URL.Query()
	service := q.Get("service")
	if service == "" {
		return fail("ERROR_NO_SERVICE")
	}
	if m.state.noNumberFor[service] {
		return fail("NO_NUMBER")
	}
	script := DefaultSmsScript()
	script.Service = service
	if c, err := strconv.Atoi(q.Get("country")); err == nil && c > 0 {
		script.Country = c
	}
	for i, s := range m.state.scripts {
		if s.Service == service {
			script = s
			m.state.scripts = append(m.state.scripts[:i], m.state.scripts[i+1:]...)
			break
		}
	}

	tzid := m.state.nextTzid()
	m.state.ops[tzid] = operation{
		Service: script.Service, Number: script.Number, Country: script.Country,
		Code: script.Code, PollsBeforeCode: script.PollsBeforeCode,
	}
	out := map[string]any{"tzid": tzid}
	if q.Has("number") { // a presence flag, like the real API
		out["number"], out["country"], out["service"], out["time"] = script.Number, script.Country, script.Service, 1200
	}
	return ok(out)
}

func (m *Server) handleGetState(r *http.Request) (int, any) {
	q := r.URL.Query()
	ids := make([]int64, 0, len(m.state.ops))
	if q.Has("tzid") {
		ids = append(ids, tzidOf(r))
	} else {
		for id := range m.state.ops {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if q.Get("orderby") == "desc" {
		sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	}

	list := []map[string]any{}
	for _, id := range ids {
		op, found := m.state.ops[id]
		if !found || op.Closed {
			continue
		}
		item := map[string]any{
			"tzid": id, "response": "TZ_NUM_WAIT", "number": op.Number, "service": op.Service,
			"country": op.Country, "sum": 0.3, "time": 600, "form": "index",
		}
		if q.Has("number") {
			item["guard_interval_remaining_seconds"] = 0
		}
		op.Polls++
		if op.Polls > op.PollsBeforeCode {
			item["response"] = "TZ_NUM_ANSWER"
			if q.Get("message_to_code") == "0" {
				item["msg"] = "Your code is " + op.Code
			} else {
				item["msg"] = op.Code
			}
		}
		m.state.ops[id] = op
		list = append(list, item)
	}
	if len(list) == 0 {
		return fail("ERROR_NO_OPERATIONS")
	}
	return http.StatusOK, list
}

func (m *Server) handleClose(r *http.Request) (int, any) {
	tzid := tzidOf(r)
	op, found := m.state.ops[tzid]
	if !found || op.Closed {
		return fail("ERROR_WRONG_TZID")
	}
	op.Closed = true
	m.state.ops[tzid] = op
	return ok(map[string]any{"tzid": tzid})
}

func (m *Server) handleRevise(r *http.Request) (int, any) {
	tzid := tzidOf(r)
	if op, found := m.state.ops[tzid]; found {
		op.Polls = 0
		m.state.ops[tzid] = op
	}
	return ok(map[string]any{"tzid": tzid}) // always succeeds, like the real API
}

var countryStats = map[string]map[string]any{
	"1":  {"name": "usa", "locale_name": "USA", "position": 1, "code": 1, "other": false, "new": false, "enabled": true},
	"44": {"name": "united_kingdom", "locale_name": "United Kingdom", "position": 3, "code": 44, "other": 0.63, "new": false, "enabled": true},
}

func countryStat(code string) map[string]any {
	out := map[string]any{"services": map[string]any{
		"service_telegram": map[string]any{"count": 10, "popular": true, "code": countryStats[code]["code"], "price": 10, "id": 1, "service": "Telegram", "slug": "telegram"},
	}}
	for k, v := range countryStats[code] {
		out[k] = v
	}
	return out
}

func (m *Server) handleTariffs(r *http.Request) (int, any) {
	country := r.URL.Query().Get("country")
	if country == "all" {
		all := map[string]any{}
		for code := range countryStats {
			all[code] = countryStat(code)
		}
		return http.StatusOK, all
	}
	if _, found := countryStats[country]; !found {
		return fail("NO_COUNTRY")
	}
	return http.StatusOK, countryStat(country)
}

var rentTariffs = map[string]map[string]any{
	"7": {
		"code": 7, "enabled": true, "name": "russia", "new": false, "position": 7, "extend": 1, "confirm": false,
		"days": map[string]float64{"1": 5, "7": 9.11}, "count": map[string]int{"1": 10, "7": 10},
	},
}

func (m *Server) handleRentTariffs(r *http.Request) (int, any) {
	if !r.URL.Query().Has("country") {
		return http.StatusOK, rentTariffs
	}
	if t, found := rentTariffs[r.URL.Query().Get("country")]; found {
		return http.StatusOK, t
	}
	return http.StatusOK, []any{} // the real API answers [] for a country without rent
}

func (m *Server) rentItem(tzid int64) map[string]any {
	r := m.state.rents[tzid]
	extension, extend := 0, any([]any{})
	if r.Extension {
		extension, extend = 1, map[string]float64{"1": 5, "7": 9.11}
	}
	return map[string]any{
		"tzid": tzid, "status": 1, "number": r.Number, "country": r.Country, "sum": "5.00",
		"days": r.Days, "hours": r.Days * 24, "time": r.Days * 24 * 60, "time_seconds": r.Days * 86400,
		"extension": extension, "extend": extend, "rent": 1, "messages": r.Messages,
		"start_time": "2026-01-01 00:00:00", "stop_time": "2026-01-02 00:00:00",
	}
}

func (m *Server) handleRentGet(r *http.Request) (int, any) {
	q := r.URL.Query()
	if _, found := rentTariffs[q.Get("country")]; !found {
		return fail("UNDEFINED_COUNTRY")
	}
	days, _ := strconv.Atoi(q.Get("days"))
	if days <= 0 {
		return fail("UNDEFINED_DAYS")
	}
	tzid := m.state.nextTzid()
	country, _ := strconv.Atoi(q.Get("country"))
	m.state.rents[tzid] = &rentOp{
		Number: "79001112233", Country: country, Days: days,
		Extension: q.Get("extension") != "0", Messages: []map[string]any{},
	}
	return ok(map[string]any{"item": m.rentItem(tzid)})
}

func (m *Server) handleRentState(r *http.Request) (int, any) {
	ids := []int64{}
	if r.URL.Query().Has("tzid") {
		if _, found := m.state.rents[tzidOf(r)]; found {
			ids = append(ids, tzidOf(r))
		}
	} else {
		for id := range m.state.rents {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	list := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		list = append(list, m.rentItem(id))
	}
	return ok(map[string]any{"list": list})
}

func (m *Server) handleRentExtend(r *http.Request) (int, any) {
	rent, found := m.state.rents[tzidOf(r)]
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if !found || !rent.Extension || days <= 0 {
		return fail("TRY_AGAIN_LATER")
	}
	rent.Days += days
	return ok(map[string]any{"item": m.rentItem(tzidOf(r))})
}

func (m *Server) handleRentClose(r *http.Request) (int, any) {
	if _, found := m.state.rents[tzidOf(r)]; !found {
		return fail("ERROR_NO_OPERATIONS")
	}
	delete(m.state.rents, tzidOf(r))
	return ok(nil)
}

func (m *Server) handleFreeList(_ *http.Request) (int, any) {
	return ok(map[string]any{
		"countries": []map[string]any{{"country": 44, "country_text": "United Kingdom", "country_original": "united_kingdom"}},
		"numbers": map[string]any{"7459608320": map[string]any{
			"country": 44, "country_original": "united_kingdom", "full_number": "447459608320", "data_humans": "1 day ago", "is_archive": false,
		}},
		"messages": map[string]any{
			"current_page": 1, "from": 1, "to": 1, "last_page": 1, "per_page": 10, "total": 1, "number": "7459608320", "country": 44,
			"data": []map[string]any{{"id": 1, "text": "Your code is 2991", "in_number": "Microsoft", "my_number": "7459608320", "code": "2991", "created_at": "2026-01-01T00:00:00.000000Z", "data_humans": "1 day ago"}},
		},
	})
}
