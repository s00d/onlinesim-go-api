package mock

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (m *Server) serve(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	path = strings.TrimSuffix(path, ".php")

	switch path {
	case "getBalance":
		m.handleBalance(w, r)
	case "getProfile":
		m.handleProfile(w, r)
	case "profile":
		m.handleSetProfile(w, r)
	case "webhook-logs":
		m.handleWebhookLogs(w, r)
	case "getPaymentHistory":
		m.writeJSON(w, map[string]any{"response": "1", "forms": map[string]any{}, "paylist": map[string]any{}})
	case "pay/createEmpty":
		m.writeJSON(w, map[string]any{"payId": 1, "params": map[string]any{}})
	case "getPrice":
		m.handlePrice(w, r)
	case "getNum":
		m.handleGetNum(w, r)
	case "getState":
		m.handleGetState(w, r)
	case "setOperationOk":
		m.handleClose(w, r, false)
	case "setOperationRevise":
		m.handleClose(w, r, true)
	case "getNumRepeat":
		m.handleGetNum(w, r)
	case "getNumbersStats":
		m.handleTariffs(w, r)
	case "getService":
		m.writeJSON(w, map[string]any{"response": "1", "service": []string{"telegram", "whatsapp"}})
	case "getServiceNumber":
		m.writeJSON(w, map[string]any{"response": "1", "number": []string{"79001234567"}})
	case "getFreeCountryList":
		m.writeJSON(w, map[string]any{
			"response": "1",
			"countries": []map[string]any{
				{"country": 7, "country_text": "Russia"},
			},
		})
	case "getFreePhoneList":
		m.writeJSON(w, map[string]any{
			"response": "1",
			"numbers": []map[string]any{
				{"number": "9001234567", "country": 7, "full_number": "+79001234567", "country_text": "Russia"},
			},
		})
	case "getFreeMessageList":
		m.writeJSON(w, map[string]any{
			"response": "1",
			"messages": map[string]any{
				"data": []map[string]any{
					{"text": "code 111", "in_number": "Telegram", "my_number": "9001234567"},
				},
			},
		})
	case "getFreeList":
		m.writeJSON(w, map[string]any{
			"countries": []map[string]any{{"country": 7, "country_text": "Russia"}},
			"numbers":   map[string]any{},
			"messages":  map[string]any{"data": []any{}, "current_page": 1},
		})
	case "rent/getRentNum":
		m.handleRentGet(w, r)
	case "rent/getRentState":
		m.handleRentState(w, r)
	case "rent/extendRentState":
		m.handleRentGet(w, r)
	case "rent/portReload", "rent/closeRentNum":
		m.writeJSON(w, map[string]any{"response": "1"})
	case "rent/tariffsRent":
		m.writeJSON(w, map[string]any{
			"7": map[string]any{
				"code": 7, "enabled": true, "name": "Russia", "new": false, "position": 1,
				"count": map[string]float64{"1": 10}, "days": map[string]float64{"1": 50}, "extend": 5,
			},
		})
	default:
		http.NotFound(w, r)
	}
}

func (m *Server) handleBalance(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.balanceError != nil {
		code := *m.state.balanceError
		m.state.balanceError = nil
		m.writeJSON(w, map[string]any{"response": code})
		return
	}
	m.writeJSON(w, map[string]any{
		"balance":  m.state.Balance,
		"zbalance": m.state.Zbalance,
		"income":   m.state.Income,
	})
}

func (m *Server) handleProfile(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeJSON(w, map[string]any{
		"response": "1",
		"profile": map[string]any{
			"id":          1,
			"name":        "mock",
			"username":    "mock",
			"email":       "mock@example.com",
			"apikey":      "mock-apikey",
			"webhook_url": m.state.WebhookURL,
		},
	})
}

func (m *Server) handleSetProfile(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var payload struct {
		Profile struct {
			WebhookURL *string `json:"webhook_url"`
		} `json:"profile"`
	}
	_ = json.Unmarshal(body, &payload)
	m.mu.Lock()
	defer m.mu.Unlock()
	if payload.Profile.WebhookURL != nil && strings.TrimSpace(*payload.Profile.WebhookURL) == "" {
		m.state.WebhookURL = nil
	} else {
		m.state.WebhookURL = payload.Profile.WebhookURL
	}
	m.saveLocked()
	m.writeJSON(w, map[string]any{"response": "1"})
}

func (m *Server) handleWebhookLogs(w http.ResponseWriter, _ *http.Request) {
	m.writeJSON(w, map[string]any{
		"data": map[string]any{
			"data":         []any{},
			"total":        0,
			"current_page": 1,
		},
	})
}

func (m *Server) handlePrice(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	price := "10"
	service := r.URL.Query().Get("service")
	for _, s := range m.state.scripts {
		if s.Service == service || service == "" {
			price = s.Price
			break
		}
	}
	pf, _ := strconv.ParseFloat(price, 64)
	m.writeJSON(w, map[string]any{"response": "1", "price": pf})
}

func (m *Server) handleGetNum(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	service := q.Get("service")
	country, _ := strconv.Atoi(q.Get("country"))
	if country == 0 {
		country = 7
	}
	wantNumber := q.Get("number") == "true"

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.state.noNumberFor {
		if s == service {
			m.writeJSON(w, map[string]any{"response": "NO_NUMBER"})
			return
		}
	}

	var script SmsScript
	found := false
	for i := range m.state.scripts {
		if m.state.scripts[i].Service == service || service == "" {
			script = m.state.scripts[i]
			m.state.scripts = append(m.state.scripts[:i], m.state.scripts[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		script = DefaultSmsScript()
		script.Service = service
		if service == "" {
			script.Service = "telegram"
		}
		script.Country = country
	}

	tzid := m.state.NextTzid
	m.state.NextTzid++
	m.state.ops[tzid] = operation{
		Service:         script.Service,
		Number:          script.Number,
		Country:         script.Country,
		Code:            script.Code,
		PollsBeforeCode: script.PollsBeforeCode,
	}
	m.saveLocked()

	out := map[string]any{
		"response": "1",
		"tzid":     tzid,
	}
	if wantNumber {
		out["number"] = script.Number
		out["country"] = script.Country
		out["service"] = script.Service
	}
	m.writeJSON(w, out)
}

func (m *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tzidStr := q.Get("tzid")

	m.mu.Lock()
	defer m.mu.Unlock()

	var list []map[string]any
	appendOp := func(tzid int64, op operation) {
		if op.Closed {
			return
		}
		item := map[string]any{
			"tzid":     tzid,
			"response": "TZ_NUM_WAIT",
			"number":   op.Number,
			"service":  op.Service,
			"country":  op.Country,
			"time":     600,
		}
		op.Polls++
		if op.Polls > op.PollsBeforeCode {
			item["msg"] = op.Code
			item["response"] = "TZ_NUM_ANSWER"
		}
		m.state.ops[tzid] = op
		list = append(list, item)
	}

	if tzidStr != "" {
		tzid, _ := strconv.ParseInt(tzidStr, 10, 64)
		if op, ok := m.state.ops[tzid]; ok {
			appendOp(tzid, op)
		}
	} else {
		for tzid, op := range m.state.ops {
			appendOp(tzid, op)
		}
	}
	m.saveLocked()
	m.writeJSON(w, list)
}

func (m *Server) handleClose(w http.ResponseWriter, r *http.Request, revise bool) {
	tzid, _ := strconv.ParseInt(r.URL.Query().Get("tzid"), 10, 64)
	m.mu.Lock()
	defer m.mu.Unlock()
	if op, ok := m.state.ops[tzid]; ok {
		if revise {
			op.Polls = 0
			op.Closed = false
			m.state.ops[tzid] = op
		} else {
			op.Closed = true
			m.state.ops[tzid] = op
		}
		m.saveLocked()
	}
	m.writeJSON(w, map[string]any{"response": "1"})
}

func (m *Server) handleTariffs(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	one := map[string]any{
		"name": "Russia", "position": 1, "code": 7, "new": false, "enabled": true,
		"services": map[string]any{
			"telegram": map[string]any{"count": 10, "popular": true, "price": 10, "id": 1, "service": "telegram", "slug": "telegram"},
		},
	}
	if country == "all" || country == "" {
		m.writeJSON(w, map[string]any{"7": one})
		return
	}
	m.writeJSON(w, one)
}

func (m *Server) handleRentGet(w http.ResponseWriter, r *http.Request) {
	country, _ := strconv.Atoi(r.URL.Query().Get("country"))
	if country == 0 {
		country = 7
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tzid := m.state.NextTzid
	m.state.NextTzid++
	op := operation{Service: "rent", Number: "+79001112233", Country: country, Code: "999111"}
	m.state.rentOps[tzid] = op
	m.state.ops[tzid] = op
	m.saveLocked()
	m.writeJSON(w, map[string]any{
		"response": "1",
		"item": map[string]any{
			"tzid": tzid, "status": 1, "country": country, "number": op.Number,
			"messages": []any{}, "rent": 1, "time": 86400, "days": 1, "hours": 24,
		},
	})
}

func (m *Server) handleRentState(w http.ResponseWriter, r *http.Request) {
	tzidStr := r.URL.Query().Get("tzid")
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []map[string]any
	for tzid, op := range m.state.rentOps {
		if tzidStr != "" {
			want, _ := strconv.ParseInt(tzidStr, 10, 64)
			if tzid != want {
				continue
			}
		}
		list = append(list, map[string]any{
			"tzid": tzid, "status": 1, "country": op.Country, "number": op.Number,
			"messages": []any{}, "rent": 1, "time": 86400,
		})
	}
	m.writeJSON(w, map[string]any{"response": "1", "list": list})
}

func (m *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
