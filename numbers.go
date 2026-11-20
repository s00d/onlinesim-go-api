package onlinesim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// NumbersAPI: numbers for receiving SMS, client.Numbers(). An operation (tzid) is one ordered number.
type NumbersAPI struct {
	do doFunc
}

// OperationStatus is the state of a number operation (Operation.Response).
type OperationStatus string

// Operation states.
const (
	TZInPool       OperationStatus = "TZ_INPOOL"      // number being assigned
	TZNumWait      OperationStatus = "TZ_NUM_WAIT"    // waiting for an SMS
	TZNumAnswer    OperationStatus = "TZ_NUM_ANSWER"  // an SMS arrived
	TZOverOK       OperationStatus = "TZ_OVER_OK"     // finished
	TZErrorNoItems OperationStatus = "ERROR_NO_ITEMS" // failed
)

// Order is the sort order of State.
type Order string

// Sort orders.
const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// Operation is a number operation (a getState item).
type Operation struct {
	Tzid int64 `json:"tzid"`
	// Response is the state; "" when the server omits it (no number could be issued).
	Response OperationStatus `json:"response"`
	Number   string          `json:"number"`
	Service  string          `json:"service"`
	Country  int             `json:"country"`
	Sum      float64         `json:"sum"`
	// Time is the number of seconds left.
	Time                  int    `json:"time"`
	Form                  string `json:"form"`
	SubscriptionPackageID int64  `json:"subscription_package_id"`
	WebhookURL            string `json:"webhook_url"`
	// Msg is empty until the first SMS arrives.
	Msg    OperationMsg `json:"msg"`
	Extend int          `json:"extend"`
	// GuardIntervalRemainingSeconds: seconds until the operation may be closed without an SMS.
	GuardIntervalRemainingSeconds int `json:"guard_interval_remaining_seconds"`
}

// OperationMsg is Operation.msg: the code (or full text with FullMessage) in Text,
// or every SMS in List when MsgList is set.
type OperationMsg struct {
	Text string
	List []OperationSMS
}

// UnmarshalJSON accepts a string, a number, a list of SMS, false or null.
func (m *OperationMsg) UnmarshalJSON(b []byte) error {
	*m = OperationMsg{}
	if t := bytes.TrimSpace(b); len(t) > 0 && t[0] == '[' {
		return json.Unmarshal(t, &m.List)
	}
	m.Text = str(b)
	return nil
}

// OperationSMS is one SMS of an operation when MsgList is set.
type OperationSMS struct {
	Service string `json:"service"`
	Msg     string `json:"msg"`
	Text    string `json:"text"`
	Code    string `json:"code"`
}

// NumberOrder is the result of GetWithNumber.
type NumberOrder struct {
	Tzid int64 `json:"tzid"`
	// Number is the full number with "+", e.g. "+447459608320".
	Number  string `json:"number"`
	Country int    `json:"country"`
	// Time is the number of seconds left.
	Time                          int    `json:"time"`
	Service                       string `json:"service"`
	Title                         string `json:"title"`
	GuardIntervalRemainingSeconds int    `json:"guard_interval_remaining_seconds"`
}

// StateOptions configures State and StateOne. The zero value is the default for every field.
type StateOptions struct {
	// FullMessage: Msg holds the full SMS text instead of the code (message_to_code=0).
	FullMessage bool
	// OrderBy is the order of State; "" means OrderAsc. Ignored by StateOne.
	OrderBy Order
	// MsgList: Msg.List holds every SMS of the operation (msg_list=1).
	MsgList bool
}

// ServiceStat is a service of a country in the tariffs.
type ServiceStat struct {
	// Count is the number of numbers in stock, -1 when the server reports "?" (unknown).
	Count   int     `json:"count"`
	Popular bool    `json:"popular"`
	Code    int     `json:"code"`
	Price   float64 `json:"price"`
	ID      int64   `json:"id"`
	// Service is the display name.
	Service string `json:"service"`
	// Slug is the service id for Get and Price.
	Slug string `json:"slug"`
}

// UnmarshalJSON accepts count "?" and numbers sent as strings.
func (s *ServiceStat) UnmarshalJSON(b []byte) error {
	type plain ServiceStat
	var v struct {
		*plain
		Count json.RawMessage `json:"count"`
		Price lenientFloat    `json:"price"`
	}
	v.plain = (*plain)(s)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.Price = float64(v.Price)
	if n, ok := num(v.Count); ok {
		s.Count = int(n)
	} else {
		s.Count = -1
	}
	return nil
}

// CountryStat is the services, prices and stock of one country.
type CountryStat struct {
	Name       string `json:"name"`
	LocaleName string `json:"locale_name"`
	Position   int    `json:"position"`
	Code       int    `json:"code"`
	// Other is the price of other services, 0 when not set.
	Other   float64 `json:"other"`
	New     bool    `json:"new"`
	Enabled bool    `json:"enabled"`
	// Services maps "service_<slug>" to the service; empty when the country has none.
	Services map[string]ServiceStat `json:"services"`
}

// UnmarshalJSON accepts services as `[]` and other as false.
func (c *CountryStat) UnmarshalJSON(b []byte) error {
	type plain CountryStat
	var v struct {
		*plain
		Other    lenientFloat        `json:"other"`
		Services phpMap[ServiceStat] `json:"services"`
	}
	v.plain = (*plain)(c)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	c.Other = float64(v.Other)
	c.Services = v.Services
	if c.Services == nil {
		c.Services = map[string]ServiceStat{}
	}
	return nil
}

// Price returns the price (USD) of one number of service in country; nil when the country has no price for it.
func (a *NumbersAPI) Price(ctx context.Context, service string, country int) (*float64, error) {
	var res struct {
		Price *float64 `json:"price"`
	}
	q := url.Values{"service": {service}, "country": {itoa(country)}}
	if err := a.do(ctx, "getPrice.php", q, nil, &res); err != nil {
		return nil, err
	}
	return res.Price, nil
}

// Get orders a number and returns the operation id (tzid). reject lists number prefixes to skip, e.g. "4470".
func (a *NumbersAPI) Get(ctx context.Context, service string, country int, reject ...string) (int64, error) {
	var res struct {
		Tzid int64 `json:"tzid"`
	}
	if err := a.do(ctx, "getNum.php", getNumQuery(service, country, reject), nil, &res); err != nil {
		return 0, err
	}
	return res.Tzid, nil
}

// GetWithNumber orders a number and returns it right away together with the operation id.
func (a *NumbersAPI) GetWithNumber(ctx context.Context, service string, country int, reject ...string) (NumberOrder, error) {
	q := getNumQuery(service, country, reject)
	q.Set("number", "1") // a presence flag on getNum: sent here only
	var res NumberOrder
	err := a.do(ctx, "getNum.php", q, nil, &res)
	return res, err
}

func getNumQuery(service string, country int, reject []string) url.Values {
	q := url.Values{"service": {service}, "country": {itoa(country)}}
	if len(reject) > 0 {
		q.Set("reject", strings.Join(reject, ","))
	}
	return q
}

// State returns all operations of the account; an empty slice when there are none.
func (a *NumbersAPI) State(ctx context.Context, opts StateOptions) ([]Operation, error) {
	order := opts.OrderBy
	if order == "" {
		order = OrderAsc
	}
	q := stateQuery(opts)
	q.Set("orderby", string(order))
	list := []Operation{}
	if err := a.do(ctx, "getState.php", q, nil, &list); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Code == "ERROR_NO_OPERATIONS" {
			return []Operation{}, nil
		}
		return nil, err
	}
	return list, nil
}

// StateOne returns one operation, or *APIError ERROR_NO_OPERATIONS when it does not exist (any more).
func (a *NumbersAPI) StateOne(ctx context.Context, tzid int64, opts StateOptions) (Operation, error) {
	q := stateQuery(opts)
	q.Set("tzid", tzidParam(tzid).Get("tzid"))
	var list []Operation
	if err := a.do(ctx, "getState.php", q, nil, &list); err != nil {
		return Operation{}, err
	}
	if len(list) == 0 {
		return Operation{}, newAPIError("ERROR_NO_OPERATIONS", "")
	}
	return list[0], nil
}

func stateQuery(opts StateOptions) url.Values {
	return url.Values{
		"message_to_code": {boolParam(!opts.FullMessage)},
		"msg_list":        {boolParam(opts.MsgList)},
		"number":          {"1"}, // adds guard_interval_remaining_seconds
	}
}

// Next switches the operation to its next SMS and keeps it open. Unknown or foreign tzid: *APIError
// ERROR_WRONG_TZID; tzid 0: ERROR_NO_TZID.
func (a *NumbersAPI) Next(ctx context.Context, tzid int64) error {
	return a.do(ctx, "setOperationRevise.php", tzidParam(tzid), nil, nil)
}

// Close finishes the operation. Without an SMS it can be closed only after its guard interval
// (NO_COMPLETE_TZID before that); every attempt locks the tzid for 5 s on the server
// (INTERVAL_CONCURRENT_REQUESTS_ERROR for a repeat). The client does not retry: retry after 5 s yourself.
func (a *NumbersAPI) Close(ctx context.Context, tzid int64) error {
	return a.do(ctx, "setOperationOk.php", tzidParam(tzid), nil, nil)
}

// Ban finishes the operation and reports the number as bad (only before any SMS arrived). After an SMS the
// server refuses with `{"response":"0"}`: *APIError with Code "0" ("Ban not applied: the operation already has an SMS").
func (a *NumbersAPI) Ban(ctx context.Context, tzid int64) error {
	q := tzidParam(tzid)
	q.Set("ban", "1")
	return a.do(ctx, "setOperationOk.php", q, nil, nil)
}

// Tariffs returns the services, prices and stock of every country, keyed by country code. No key needed.
func (a *NumbersAPI) Tariffs(ctx context.Context) (map[string]CountryStat, error) {
	var res phpMap[CountryStat]
	if err := a.do(ctx, "getNumbersStats.php", url.Values{"country": {"all"}}, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// TariffsOne returns the services, prices and stock of one country. No key needed.
func (a *NumbersAPI) TariffsOne(ctx context.Context, country int) (CountryStat, error) {
	var res CountryStat
	err := a.do(ctx, "getNumbersStats.php", url.Values{"country": {itoa(country)}}, nil, &res)
	return res, err
}
