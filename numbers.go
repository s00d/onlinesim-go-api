package onlinesim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// NumbersAPI covers temporary SMS number operations.
type NumbersAPI struct{ c *Client }

// GetNumberParams configures a number order.
type GetNumberParams struct {
	Service   string
	Country   int
	Reject    []int
	Extension bool
	Number    bool // when true, ask API to return number in response
}

// StateParams configures getState polling.
type StateParams struct {
	// MessageToCode controls code extraction. nil defaults to 1; use IntPtr(0) for full message.
	MessageToCode *int
	OrderBy       string // ASC / DESC
	MsgList       bool
	Clean         bool
	Repeat        bool
	Tzid          int64
}

// IntPtr returns a pointer to v (for optional int params).
func IntPtr(v int) *int { return &v }

// NumberWithTz is returned by GetWithNumber.
type NumberWithTz struct {
	Tzid                           int64  `json:"tzid"`
	Number                         string `json:"number"`
	Country                        int    `json:"country"`
	Time                           *int64 `json:"time,omitempty"`
	Service                        string `json:"service,omitempty"`
	Title                          string `json:"title,omitempty"`
	Response                       string `json:"response,omitempty"`
	GuardIntervalRemainingSeconds  *int64 `json:"guard_interval_remaining_seconds,omitempty"`
}

// StateOne is a single operation from getState.
type StateOne struct {
	Tzid                          int64           `json:"tzid"`
	Response                      string          `json:"response"`
	Number                        string          `json:"number"`
	Service                       string          `json:"service"`
	Time                          int64           `json:"time"`
	Msg                           json.RawMessage `json:"msg,omitempty"`
	Country                       int             `json:"country"`
	Sum                           float64         `json:"sum,omitempty"`
	Form                          json.RawMessage `json:"form,omitempty"`
	ResponseText                  string          `json:"response_text,omitempty"`
	Title                         string          `json:"title,omitempty"`
	GuardIntervalRemainingSeconds *int64          `json:"guard_interval_remaining_seconds,omitempty"`
}

// MsgString extracts a string code/message from Msg.
func (s StateOne) MsgString() string {
	if len(s.Msg) == 0 || string(s.Msg) == "null" {
		return ""
	}
	var str string
	if err := json.Unmarshal(s.Msg, &str); err == nil {
		return str
	}
	var num json.Number
	if err := json.Unmarshal(s.Msg, &num); err == nil {
		return num.String()
	}
	var arr []struct {
		Service string `json:"service"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(s.Msg, &arr); err == nil && len(arr) > 0 {
		return arr[0].Msg
	}
	return strings.Trim(string(s.Msg), `"`)
}

// TariffService is a service row inside country tariffs.
type TariffService struct {
	Count   any     `json:"count"`
	Popular bool    `json:"popular"`
	Code    int     `json:"code"`
	Price   float64 `json:"price"`
	ID      any     `json:"id"`
	Service any     `json:"service"`
	Slug    any     `json:"slug"`
}

// UnmarshalJSON accepts live price as number or number_format string.
func (s *TariffService) UnmarshalJSON(data []byte) error {
	type alias TariffService
	var raw struct {
		alias
		Price json.RawMessage `json:"price"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = TariffService(raw.alias)
	price, err := parseFlexFloat(raw.Price)
	if err != nil {
		return err
	}
	s.Price = price
	return nil
}

// TariffCountryOne is tariffs for one country.
type TariffCountryOne struct {
	Name     string                   `json:"name"`
	Position int                      `json:"position"`
	Code     int                      `json:"code"`
	Other    any                      `json:"other"`
	New      bool                     `json:"new"`
	Enabled  bool                     `json:"enabled"`
	Services map[string]TariffService `json:"services"`
}

// UnmarshalJSON accepts services as object map or empty JSON array.
func (c *TariffCountryOne) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name     string          `json:"name"`
		Position int             `json:"position"`
		Code     int             `json:"code"`
		Other    any             `json:"other"`
		New      bool            `json:"new"`
		Enabled  bool            `json:"enabled"`
		Services json.RawMessage `json:"services"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Name = raw.Name
	c.Position = raw.Position
	c.Code = raw.Code
	c.Other = raw.Other
	c.New = raw.New
	c.Enabled = raw.Enabled
	c.Services = map[string]TariffService{}
	if len(raw.Services) == 0 || string(raw.Services) == "null" {
		return nil
	}
	trimmed := bytes.TrimSpace(raw.Services)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []TariffService
		if err := json.Unmarshal(raw.Services, &list); err != nil {
			return err
		}
		for i, svc := range list {
			key := serviceKey(svc, i)
			if key != "" {
				c.Services[key] = svc
			}
		}
		return nil
	}
	return json.Unmarshal(raw.Services, &c.Services)
}

func serviceKey(svc TariffService, i int) string {
	switch v := svc.Slug.(type) {
	case string:
		if v != "" {
			return v
		}
	}
	switch v := svc.Service.(type) {
	case string:
		if v != "" {
			return v
		}
	}
	return strconv.Itoa(i)
}

func parseFlexFloat(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return 0, nil
		}
		return strconv.ParseFloat(s, 64)
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.Float64()
	}
	return 0, fmt.Errorf("expected number, got %s", string(raw))
}

// Price returns the price for a service in a country.
func (a *NumbersAPI) Price(ctx context.Context, service string, country int) (float64, error) {
	if country == 0 {
		country = DefaultCountry
	}
	var resp struct {
		Response any             `json:"response"`
		Price    json.RawMessage `json:"price"`
	}
	err := a.c.tr.GetJSON(ctx, "getPrice", map[string]string{
		"service": service,
		"country": strconv.Itoa(country),
	}, true, &resp)
	if err != nil {
		return 0, err
	}
	if len(resp.Price) == 0 || string(resp.Price) == "null" {
		return 0, nil
	}
	return parseFlexFloat(resp.Price)
}

// Get orders a number and returns tzid.
func (a *NumbersAPI) Get(ctx context.Context, p GetNumberParams) (int64, error) {
	res, err := a.getRaw(ctx, p)
	if err != nil {
		return 0, err
	}
	return res.Tzid, nil
}

// GetWithNumber orders a number and returns tzid + phone metadata.
func (a *NumbersAPI) GetWithNumber(ctx context.Context, p GetNumberParams) (NumberWithTz, error) {
	p.Number = true
	return a.getRaw(ctx, p)
}

func (a *NumbersAPI) getRaw(ctx context.Context, p GetNumberParams) (NumberWithTz, error) {
	if p.Country == 0 {
		p.Country = DefaultCountry
	}
	service := strings.TrimPrefix(p.Service, "service_")
	params := map[string]string{
		"service":   service,
		"country":   strconv.Itoa(p.Country),
		"extension": strconv.FormatBool(p.Extension),
	}
	if p.Number {
		params["number"] = "true"
	}
	if len(p.Reject) > 0 {
		parts := make([]string, len(p.Reject))
		for i, v := range p.Reject {
			parts[i] = strconv.Itoa(v)
		}
		params["reject"] = strings.Join(parts, ",")
	}
	var resp struct {
		Response any    `json:"response"`
		Tzid     int64  `json:"tzid"`
		Number   string `json:"number"`
		Country  int    `json:"country"`
		Time     *int64 `json:"time"`
		Service  string `json:"service"`
		Title    string `json:"title"`
	}
	if err := a.c.tr.GetJSON(ctx, "getNum", params, true, &resp); err != nil {
		return NumberWithTz{}, err
	}
	country := resp.Country
	if country == 0 {
		country = p.Country
	}
	return NumberWithTz{
		Tzid:    resp.Tzid,
		Number:  resp.Number,
		Country: country,
		Time:    resp.Time,
		Service: resp.Service,
		Title:   resp.Title,
	}, nil
}

// State lists active operations.
func (a *NumbersAPI) State(ctx context.Context, p StateParams) ([]StateOne, error) {
	p = p.withDefaults()
	params := stateQuery(p)
	if p.Repeat {
		params["type"] = "repeat"
	} else {
		params["type"] = "index"
	}
	var list []StateOne
	if err := a.c.tr.GetJSON(ctx, "getState", params, true, &list); err != nil {
		if errors.Is(err, ErrNoOperations) {
			return []StateOne{}, nil
		}
		return nil, err
	}
	return list, nil
}

// StateOne returns a single operation by tzid.
func (a *NumbersAPI) StateOne(ctx context.Context, tzid int64, p StateParams) (StateOne, error) {
	p = p.withDefaults()
	p.Tzid = tzid
	params := stateQuery(p)
	if p.Repeat {
		params["type"] = "repeat"
	} else {
		params["type"] = "index"
	}
	params["tzid"] = strconv.FormatInt(tzid, 10)
	var list []StateOne
	if err := a.c.tr.GetJSON(ctx, "getState", params, true, &list); err != nil {
		if errors.Is(err, ErrNoOperations) {
			return StateOne{}, fmt.Errorf("no operation for tzid %d", tzid)
		}
		return StateOne{}, err
	}
	if len(list) == 0 {
		return StateOne{}, fmt.Errorf("no operation for tzid %d", tzid)
	}
	return list[0], nil
}

func (p StateParams) withDefaults() StateParams {
	// Match JS defaults when caller passes a zero-value struct.
	if p.MessageToCode == nil && !p.MsgList && !p.Clean && !p.Repeat && p.OrderBy == "" && p.Tzid == 0 {
		p.MsgList = true
		p.Clean = true
	}
	return p
}

func stateQuery(p StateParams) map[string]string {
	msgToCode := 1
	if p.MessageToCode != nil {
		msgToCode = *p.MessageToCode
	}
	order := p.OrderBy
	if order == "" {
		order = "ASC"
	}
	msgList := "0"
	if p.MsgList {
		msgList = "1"
	}
	clean := "0"
	if p.Clean {
		clean = "1"
	}
	repeat := "0"
	if p.Repeat {
		repeat = "1"
	}
	return map[string]string{
		"message_to_code": strconv.Itoa(msgToCode),
		"orderby":         order,
		"msg_list":        msgList,
		"clean":           clean,
		"repeat":          repeat,
	}
}

// Next revises the operation to wait for another SMS.
func (a *NumbersAPI) Next(ctx context.Context, tzid int64) error {
	var resp struct {
		Response any `json:"response"`
	}
	return a.c.tr.GetJSON(ctx, "setOperationRevise", map[string]string{
		"tzid": strconv.FormatInt(tzid, 10),
	}, true, &resp)
}

// Close completes the operation.
func (a *NumbersAPI) Close(ctx context.Context, tzid int64) error {
	return a.setOperationOk(ctx, tzid, false)
}

// Ban completes the operation and bans the number (setOperationOk?ban=1).
func (a *NumbersAPI) Ban(ctx context.Context, tzid int64) error {
	return a.setOperationOk(ctx, tzid, true)
}

func (a *NumbersAPI) setOperationOk(ctx context.Context, tzid int64, ban bool) error {
	var resp struct {
		Response any `json:"response"`
	}
	params := map[string]string{
		"tzid": strconv.FormatInt(tzid, 10),
	}
	if ban {
		params["ban"] = "1"
	}
	return a.c.tr.GetJSON(ctx, "setOperationOk", params, true, &resp)
}

// Repeat starts a repeated reception for a number.
//
// Note: production OnlineSim may not expose getNumRepeat (demo-only in some
// deployments). Prefer a fresh Get when unsure.
func (a *NumbersAPI) Repeat(ctx context.Context, service string, number int64) (int64, error) {
	var resp struct {
		Response any   `json:"response"`
		Tzid     int64 `json:"tzid"`
	}
	err := a.c.tr.GetJSON(ctx, "getNumRepeat", map[string]string{
		"service": service,
		"number":  strconv.FormatInt(number, 10),
	}, true, &resp)
	return resp.Tzid, err
}

// Tariffs returns tariffs for all countries.
func (a *NumbersAPI) Tariffs(ctx context.Context) (map[string]TariffCountryOne, error) {
	out := map[string]TariffCountryOne{}
	err := a.c.tr.GetJSON(ctx, "getNumbersStats", map[string]string{"country": "all"}, true, &out)
	return out, err
}

// TariffsOne returns tariffs for one country.
func (a *NumbersAPI) TariffsOne(ctx context.Context, country int) (TariffCountryOne, error) {
	if country == 0 {
		country = DefaultCountry
	}
	var out TariffCountryOne
	err := a.c.tr.GetJSON(ctx, "getNumbersStats", map[string]string{
		"country": strconv.Itoa(country),
	}, true, &out)
	return out, err
}

// Service returns available service slugs.
func (a *NumbersAPI) Service(ctx context.Context) ([]string, error) {
	var resp struct {
		Response any      `json:"response"`
		Service  []string `json:"service"`
	}
	err := a.c.tr.GetJSON(ctx, "getService", map[string]string{}, true, &resp)
	return resp.Service, err
}

// ServiceNumber returns numbers available for repeated reception.
func (a *NumbersAPI) ServiceNumber(ctx context.Context, service string) ([]string, error) {
	var resp struct {
		Response any      `json:"response"`
		Number   []string `json:"number"`
	}
	err := a.c.tr.GetJSON(ctx, "getServiceNumber", map[string]string{"service": service}, true, &resp)
	return resp.Number, err
}
