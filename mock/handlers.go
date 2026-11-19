package mock

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// The mock's routes: endpoint path (relative to `/api/`) → handler. A handler gets the state and the request
// and returns the JSON body the real backend would send (HTTP 200). Shapes follow the OpenAPI spec and the
// live captures: `response` as int or string per endpoint, decimal-string money, `[]` for empty maps.
// Auth, queued failures, 456 and 404 are the transport's job (mock.go), not the handlers'.

type obj = map[string]any

type handler func(s *state, r *Request) any

var routes = map[string]handler{
	"getPrice.php":             getPrice,
	"getNum.php":               getNum,
	"getState.php":             getState,
	"setOperationRevise.php":   setOperationRevise,
	"setOperationOk.php":       setOperationOk,
	"getNumbersStats.php":      getNumbersStats,
	"rent/getRentNum.php":      getRentNum,
	"rent/getRentState.php":    getRentState,
	"rent/extendRentState.php": extendRentState,
	"rent/closeRentNum.php":    closeRentNum,
	"rent/tariffsRent.php":     tariffsRent,
	"getBalance.php":           getBalance,
	"getProfile.php":           getProfile,
	"profile.php":              profile,
	"webhook-logs":             webhookLogs,
	"getPaymentHistory.php":    getPaymentHistory,
	"getFreeList":              getFreeList,
}

// publicPaths answer without a key.
var publicPaths = map[string]bool{"getNumbersStats.php": true, "rent/tariffsRent.php": true, "getFreeList": true}

var countryNames = map[int][2]string{
	7:  {"russia", "Russia"},
	44: {"united_kingdom", "United Kingdom"},
}

func countryName(code int) (name, locale string) {
	if n, ok := countryNames[code]; ok {
		return n[0], n[1]
	}
	return fmt.Sprintf("country_%d", code), fmt.Sprintf("Country %d", code)
}

func apiError(code string) any { return obj{"response": code} }

// flag is a presence flag: set and not "0".
func flag(r *Request, key string) bool {
	v, ok := r.Query[key]
	return ok && v != "0"
}

func intParam(r *Request, key string, fallback int) int {
	n, err := strconv.Atoi(r.Query[key])
	if err != nil {
		return fallback
	}
	return n
}

func tzidParam(r *Request, fallback int64) int64 {
	n, err := strconv.ParseInt(r.Query["tzid"], 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

// decimal is USD as a decimal string, as the backend sends `balance` (3 digits) and rent `sum` (2 digits).
func decimal(mills int64, digits int) string {
	return strconv.FormatFloat(fromMills(mills), 'f', digits, 64)
}

// paginate is a Laravel paginator over items.
func paginate[T any](items []T, page, perPage int, path string) obj {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	total := len(items)
	lastPage := max(1, (total+perPage-1)/perPage)
	from, to := min((page-1)*perPage, total), min(page*perPage, total)
	data := append([]T{}, items[from:to]...)
	pageURL := func(p int) string { return fmt.Sprintf("https://onlinesim.host/api/%s?page=%d", path, p) }
	out := obj{
		"current_page": page, "data": data, "first_page_url": pageURL(1), "from": nil, "last_page": lastPage,
		"last_page_url": pageURL(lastPage), "links": []any{}, "next_page_url": nil,
		"path": "https://onlinesim.host/api/" + path, "per_page": perPage, "prev_page_url": nil, "to": nil, "total": total,
	}
	if len(data) > 0 {
		out["from"], out["to"] = from+1, to
	}
	if page < lastPage {
		out["next_page_url"] = pageURL(page + 1)
	}
	if page > 1 {
		out["prev_page_url"] = pageURL(page - 1)
	}
	return out
}

// ----- numbers -----

func msgOf(sms SMS, toCode bool) string {
	if toCode && sms.Code != "" {
		return sms.Code
	}
	return sms.Text
}

func operationView(s *state, op *Operation, r *Request) obj {
	toCode := flag(r, "message_to_code")
	open := op.Status != onlinesim.TZOverOK
	view := obj{
		"country": op.Country, "sum": op.Sum, "service": op.Service, "number": op.Number,
		"subscription_package_id": nil, "webhook_url": s.webhookURL, "response": op.Status, "tzid": op.Tzid,
		"time": 0, "form": "index",
	}
	if open {
		view["time"] = operationTime
	}
	if flag(r, "msg_list") {
		if len(op.Messages) > 0 {
			list := make([]obj, len(op.Messages))
			for i, sms := range op.Messages {
				list[i] = obj{"service": sms.Service, "msg": msgOf(sms, toCode)}
			}
			view["msg"] = list
		}
	} else if op.Cursor < len(op.Messages) {
		view["msg"] = msgOf(op.Messages[op.Cursor], toCode)
	}
	if _, ok := r.Query["number"]; ok {
		guard := 0
		if open && len(op.Messages) == 0 {
			guard = s.guardInterval
		}
		view["guard_interval_remaining_seconds"] = guard
	}
	return view
}

func getPrice(s *state, r *Request) any {
	prices, ok := s.prices[intParam(r, "country", 7)]
	if !ok {
		return apiError("UNDEFINED_COUNTRY")
	}
	var price any
	if p, ok := prices[r.Query["service"]]; ok {
		price = p
	}
	return obj{"response": "1", "price": price}
}

func getNum(s *state, r *Request) any {
	country := intParam(r, "country", 7)
	service := r.Query["service"]
	prices, ok := s.prices[country]
	if !ok {
		return apiError("UNDEFINED_COUNTRY")
	}
	price, ok := prices[service]
	if !ok {
		return apiError("ERROR_NO_SERVICE")
	}
	cost := toMills(price)
	if s.available() < cost {
		return apiError("WARNING_LOW_BALANCE")
	}

	digits := strconv.Itoa(country) + shortNumber(s.seq+1)
	for _, prefix := range strings.Split(r.Query["reject"], ",") {
		if prefix != "" && strings.HasPrefix(digits, prefix) {
			return apiError("NO_NUMBER")
		}
	}

	tzid := s.nextID()
	s.frozen += cost
	op := &Operation{Tzid: tzid, Service: service, Country: country, Number: "+" + digits, Sum: price, Status: onlinesim.TZNumWait}
	s.operations[tzid] = op
	if _, ok := r.Query["number"]; !ok {
		return obj{"response": 1, "tzid": tzid}
	}
	return obj{"response": 1, "tzid": tzid, "number": op.Number, "country": country, "time": operationTime, "service": service}
}

func getState(s *state, r *Request) any {
	var ops []*Operation
	if _, ok := r.Query["tzid"]; ok {
		if op, found := s.operations[tzidParam(r, -1)]; found {
			ops = append(ops, op)
		}
	} else {
		ops = s.sortedOperations()
	}
	for _, op := range ops {
		s.poll(op.Tzid)
	}
	if len(ops) == 0 {
		return apiError("ERROR_NO_OPERATIONS")
	}
	if r.Query["orderby"] == "desc" {
		sort.Slice(ops, func(i, j int) bool { return ops[i].Tzid > ops[j].Tzid })
	}
	list := make([]obj, len(ops))
	for i, op := range ops {
		list[i] = operationView(s, op, r)
	}
	return list
}

func setOperationRevise(s *state, r *Request) any {
	tzid := tzidParam(r, 0)
	// Always succeeds, even for unknown tzids, like the backend.
	if op, ok := s.operations[tzid]; ok && op.Status != onlinesim.TZOverOK {
		op.Cursor = len(op.Messages)
		op.Status = onlinesim.TZNumWait
	}
	return obj{"response": 1, "tzid": tzid}
}

func setOperationOk(s *state, r *Request) any {
	tzid := tzidParam(r, 0)
	op, ok := s.operations[tzid]
	if !ok || op.Status == onlinesim.TZOverOK {
		return apiError("ERROR_WRONG_TZID")
	}
	received := len(op.Messages) > 0
	if flag(r, "ban") && received {
		return obj{"response": "0"}
	}
	if !received && s.guardInterval > 0 {
		return apiError("NO_COMPLETE_TZID")
	}
	cost := toMills(op.Sum)
	s.frozen -= cost
	if received {
		s.balance -= cost // spent; without an SMS the frozen price is refunded
	}
	op.Status = onlinesim.TZOverOK
	return obj{"response": 1, "tzid": tzid}
}

func countryStat(s *state, code int) obj {
	name, locale := countryName(code)
	prices := s.prices[code]
	slugs := make([]string, 0, len(prices))
	for slug := range prices {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs) // service ids follow the slug order: Go maps have none
	var services any = []any{}
	if len(slugs) > 0 {
		m := obj{}
		for i, slug := range slugs {
			m["service_"+slug] = obj{"count": 100, "popular": false, "code": code, "price": prices[slug], "id": i + 1, "service": slug, "slug": slug}
		}
		services = m
	}
	return obj{
		"name": name, "position": code, "code": code, "other": false, "new": false, "enabled": true,
		"services": services, "locale_name": locale,
	}
}

func getNumbersStats(s *state, r *Request) any {
	if r.Query["country"] == "all" {
		if len(s.prices) == 0 {
			return []any{}
		}
		all := obj{}
		for code := range s.prices {
			all[strconv.Itoa(code)] = countryStat(s, code)
		}
		return all
	}
	code := intParam(r, "country", 7)
	if _, ok := s.prices[code]; !ok {
		return apiError("NO_COUNTRY")
	}
	return countryStat(s, code)
}

// ----- rent -----

func dayMap(days map[int]float64) obj {
	out := obj{}
	for d, price := range days {
		out[strconv.Itoa(d)] = price
	}
	return out
}

func rentItem(s *state, rent *Rent, r *Request) obj {
	tariff, ok := s.rentTariffs[rent.Country]
	extendable := rent.Extension && ok && tariff.Extend == 1
	messages := make([]obj, 0, len(rent.Messages))
	for i := len(rent.Messages) - 1; i >= 0; i-- { // newest first
		m := rent.Messages[i]
		messages = append(messages, obj{"id": m.ID, "service": m.Service, "text": m.Text, "code": m.Code, "created_at": m.CreatedAt})
	}
	start := rent.Tzid
	extension := 0
	if rent.Extension {
		extension = 1
	}
	var extend any = []any{}
	dayExtend := 0.0
	if extendable {
		extend, dayExtend = dayMap(tariff.Days), tariff.Days[1]
	}
	item := obj{
		"status": 1, "extension": extension, "messages": messages, "webhook_url": s.webhookURL,
		"sum": decimal(toMills(rent.Sum), 2), "country": rent.Country,
		"start_time": stamp(start), "stop_time": stamp(start + int64(rent.Days)*86_400), "checked_time": stamp(start),
		"number": rent.Number, "days_autoextend": 0, "rent": 1, "tzid": rent.Tzid,
		"time": rent.Days * 24 * 60, "days": rent.Days, "hours": rent.Days * 24,
		"extend": extend, "checked": true, "reload": nil, "day_extend": dayExtend, "m_ext": false, "freeze": false,
	}
	if _, ok := r.Query["number"]; ok {
		item["time_seconds"] = rent.Days * 86_400
	}
	return item
}

func getRentNum(s *state, r *Request) any {
	country := intParam(r, "country", 7)
	tariff, ok := s.rentTariffs[country]
	if !ok {
		return apiError("NO_COUNTRY")
	}
	days := intParam(r, "days", 1)
	price, ok := tariff.Days[days]
	if !ok {
		return apiError("UNDEFINED_DAYS")
	}
	if s.available() < toMills(price) {
		return apiError("WARNING_LOW_BALANCE")
	}
	tzid := s.nextID()
	s.balance -= toMills(price)
	rent := &Rent{
		Tzid: tzid, Country: country, Number: shortNumber(tzid), Days: days, Sum: price,
		// `extension` defaults to 1 on the server; `0` makes the rent non-extendable for good.
		Extension: r.Query["extension"] != "0",
	}
	s.rents[tzid] = rent
	return obj{"response": 1, "item": rentItem(s, rent, r)}
}

func getRentState(s *state, r *Request) any {
	var rents []*Rent
	if _, ok := r.Query["tzid"]; ok {
		if rent, found := s.rents[tzidParam(r, -1)]; found {
			rents = append(rents, rent)
		}
	} else {
		rents = s.sortedRents()
	}
	list := make([]obj, len(rents))
	for i, rent := range rents {
		s.poll(rent.Tzid)
		list[i] = rentItem(s, rent, r)
	}
	return obj{"response": 1, "list": list}
}

func extendRentState(s *state, r *Request) any {
	rent, ok := s.rents[tzidParam(r, 0)]
	if !ok {
		return apiError("ERROR_WRONG_TZID")
	}
	tariff, ok := s.rentTariffs[rent.Country]
	if !rent.Extension || !ok || tariff.Extend != 1 {
		return apiError("TRY_AGAIN_LATER")
	}
	days := intParam(r, "days", 1)
	price, ok := tariff.Days[days]
	if !ok {
		return apiError("UNDEFINED_DAYS")
	}
	if s.available() < toMills(price) {
		return apiError("WARNING_LOW_BALANCE")
	}
	s.balance -= toMills(price)
	rent.Days += days
	rent.Sum = fromMills(toMills(rent.Sum) + toMills(price))
	return obj{"response": 1, "item": rentItem(s, rent, r)}
}

func closeRentNum(s *state, r *Request) any {
	tzid := tzidParam(r, 0)
	if _, ok := s.rents[tzid]; !ok {
		return apiError("ERROR_WRONG_TZID")
	}
	delete(s.rents, tzid) // no refund
	delete(s.scheduled, tzid)
	return obj{"response": "1"}
}

func tariffsRent(s *state, r *Request) any {
	tariff := func(code int) any {
		t, ok := s.rentTariffs[code]
		if !ok {
			return []any{}
		}
		name, _ := countryName(code)
		count := obj{}
		for d := range t.Days {
			count[strconv.Itoa(d)] = 100
		}
		return obj{
			"code": code, "enabled": true, "name": name, "new": false, "position": code, "count": count,
			"days": dayMap(t.Days), "extend": t.Extend, "confirm": false, "ext_counts": count,
		}
	}
	if _, ok := r.Query["country"]; ok {
		return tariff(intParam(r, "country", 0))
	}
	if len(s.rentTariffs) == 0 {
		return []any{}
	}
	all := obj{}
	for code := range s.rentTariffs {
		all[strconv.Itoa(code)] = tariff(code)
	}
	return all
}

// ----- user -----

func getBalance(s *state, r *Request) any {
	res := obj{"response": "1", "balance": decimal(s.balance, 3), "zbalance": fromMills(s.frozen)}
	if _, ok := r.Query["income"]; ok {
		res["income"], res["income_usd"] = 0, "0.000"
	}
	return res
}

func getProfile(s *state, _ *Request) any {
	return obj{"response": "1", "profile": obj{
		"id": userID, "hash_id": "mock", "name": "mock@example.com", "username": "mock@example.com",
		"email": "mock@example.com", "email_verified_at": stamp(0), "active": "true", "apikey": s.apiKey,
		"api_access": true, "is_api": true, "number": nil, "country": nil, "locale": "en", "currency": "USD",
		"number_region": nil, "number_country": nil, "number_reject": []any{}, "ugroup": 1, "verify": 0, "block": 0,
		"resetCount": nil, "created_at": stamp(0), "notification_at": nil, "first_login": false, "sub": 0,
		"disable": false, "webhook_url": s.webhookURL, "is_project_subscription": false,
	}}
}

func isHTTPURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

func profile(s *state, r *Request) any {
	body, _ := r.Body.(obj)
	p, ok := body["profile"].(obj)
	if r.Method != "POST" || !ok {
		return apiError("ERROR_PARAMS")
	}
	raw, present := p["webhook_url"]
	if raw != nil {
		u, isString := raw.(string)
		if !isString || !isHTTPURL(u) {
			// A validation error is HTTP 200 with the field messages in `response`.
			return obj{"response": obj{"webhook_url": []string{"The webhook url must be a valid URL."}}}
		}
		s.webhookURL = &u
	} else if present {
		s.webhookURL = nil
	}
	return obj{"response": "1", "user": obj{"id": userID, "webhook_url": s.webhookURL}}
}

func webhookLogs(s *state, r *Request) any {
	logs := make([]webhookLog, len(s.webhookLogs))
	for i, l := range s.webhookLogs { // newest first
		logs[len(logs)-1-i] = l
	}
	return obj{"response": "1", "data": paginate(logs, intParam(r, "page", 1), 10, "webhook-logs")}
}

func getPaymentHistory(_ *state, r *Request) any {
	return obj{
		"response": "1", "orders": paginate([]any{}, intParam(r, "page", 1), 10, "getPaymentHistory.php"),
		"paylist": []any{}, "domains": []any{}, "currency": obj{"USD": 1},
	}
}

// ----- free -----

type freeCountry struct {
	name, text string
	numbers    map[string][][2]string // short number → [sender, text]
}

var freeData = map[int]freeCountry{
	44: {name: "united_kingdom", text: "United Kingdom", numbers: map[string][][2]string{
		"7459608320": {
			{"Amazon", "Amazon: Use 195895 to reset your password."},
			{"Microsoft", "Microsoft access code: 2991"},
		},
	}},
	7: {name: "russia", text: "Russia", numbers: map[string][][2]string{
		"9001234567": {{"Telegram", "Telegram code: 51423"}},
	}},
}

func getFreeList(_ *state, r *Request) any {
	codes := sortedKeys(freeData)
	country := intParam(r, "country", 44)
	data, ok := freeData[country]
	if !ok {
		country = codes[0]
		data = freeData[country]
	}
	shorts := make([]string, 0, len(data.numbers))
	numbers := obj{}
	for n := range data.numbers {
		shorts = append(shorts, n)
		numbers[n] = obj{
			"country": country, "country_original": data.name, "data_humans": "1 day ago",
			"full_number": strconv.Itoa(country) + n, "is_archive": false,
		}
	}
	sort.Strings(shorts)
	number := r.Query["number"]
	if _, ok := data.numbers[number]; !ok {
		number = shorts[0]
	}
	messages := make([]obj, 0, len(data.numbers[number]))
	for i, m := range data.numbers[number] {
		code := extractCode(m[1])
		if code == "" {
			code = "-"
		}
		messages = append(messages, obj{
			"id": i + 1, "text": m[1], "in_number": m[0], "my_number": number,
			"created_at": stamp(int64(i)), "data_humans": "1 day ago", "code": code,
		})
	}
	countries := make([]obj, len(codes))
	for i, c := range codes {
		countries[i] = obj{"country": c, "country_text": freeData[c].text, "country_original": freeData[c].name}
	}
	page := paginate(messages, intParam(r, "page", 1), intParam(r, "count", 10), "getFreeList")
	page["number"], page["country"] = number, country
	return obj{"response": 1, "countries": countries, "numbers": numbers, "messages": page}
}
