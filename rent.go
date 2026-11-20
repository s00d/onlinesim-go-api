package onlinesim

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
)

// RentAPI: rented numbers, client.Rent(). A rent keeps a number for days and receives every SMS sent to it.
type RentAPI struct {
	do doFunc
}

// RentSMS is one SMS of a rent.
type RentSMS struct {
	ID int64 `json:"id"`
	// Service is the sender. "notify": a notice about a prohibited sender (does not confirm the rent);
	// "init": the record of a rent confirmed at once.
	Service string `json:"service"`
	Text    string `json:"text"`
	// Code is the code found in the text, "" when none. The server sends the integer 0 for a "notify" SMS;
	// it is normalised to "0".
	Code      string `json:"code"`
	CreatedAt string `json:"created_at"`
}

// UnmarshalJSON accepts code as a string or a number (the integer 0 of a "notify" SMS becomes "0").
func (s *RentSMS) UnmarshalJSON(b []byte) error {
	type plain RentSMS
	var v struct {
		*plain
		Code json.RawMessage `json:"code"`
	}
	v.plain = (*plain)(s)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	s.Code = str(v.Code)
	return nil
}

// RentItem is a rent operation.
type RentItem struct {
	Tzid    int64  `json:"tzid"`
	Status  int    `json:"status"`
	Number  string `json:"number"`
	Country int    `json:"country"`
	// Sum is the cost of the rent incl. extensions, USD (the server sends a number in Get/Extend and a decimal
	// string in State). Reserved (frozen in zbalance) while Status is 0, charged once the first SMS confirms it.
	Sum   float64 `json:"sum"`
	Days  int     `json:"days"`
	Hours int     `json:"hours"`
	// Time is the number of minutes left.
	Time int `json:"time"`
	// TimeSeconds is the number of seconds left.
	TimeSeconds                   int64 `json:"time_seconds"`
	GuardIntervalRemainingSeconds int   `json:"guard_interval_remaining_seconds"`
	// Extension: 0 not extendable, 1 extendable, 2 auto-extension.
	Extension      int    `json:"extension"`
	DaysAutoextend int    `json:"days_autoextend"`
	Rent           int    `json:"rent"`
	StartTime      string `json:"start_time"`
	StopTime       string `json:"stop_time"`
	CheckedTime    string `json:"checked_time"`
	Checked        bool   `json:"checked"`
	Freeze         bool   `json:"freeze"`
	Reload         int    `json:"reload"`
	MExt           bool   `json:"m_ext"`
	// DayExtend is the price of one day of extension; 0 when not configured.
	DayExtend float64 `json:"day_extend"`
	// Extend maps days to the extension price; empty when not extendable.
	Extend     map[string]float64 `json:"extend"`
	WebhookURL string             `json:"webhook_url"`
	// Messages are the SMS, newest first.
	Messages []RentSMS `json:"messages"`
}

// UnmarshalJSON accepts sum as a number or a decimal string and extend as `[]`.
func (r *RentItem) UnmarshalJSON(b []byte) error {
	type plain RentItem
	var v struct {
		*plain
		Sum    lenientFloat    `json:"sum"`
		Extend phpMap[float64] `json:"extend"`
	}
	v.plain = (*plain)(r)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	r.Sum = float64(v.Sum)
	r.Extend = v.Extend
	if r.Extend == nil {
		r.Extend = map[string]float64{}
	}
	if r.Messages == nil {
		r.Messages = []RentSMS{}
	}
	return nil
}

// RentTariff is the rent prices of one country.
type RentTariff struct {
	Code     int     `json:"code"`
	Name     string  `json:"name"`
	Enabled  bool    `json:"enabled"`
	New      bool    `json:"new"`
	Position int     `json:"position"`
	Extend   float64 `json:"extend"`
	Confirm  bool    `json:"confirm"`
	// Days maps rent days to the price, USD.
	Days map[string]float64 `json:"days"`
	// Count maps rent days to the numbers available.
	Count     map[string]int `json:"count"`
	ExtCounts map[string]int `json:"ext_counts"`
}

// UnmarshalJSON accepts the maps as `[]`.
func (t *RentTariff) UnmarshalJSON(b []byte) error {
	type plain RentTariff
	var v struct {
		*plain
		Days      phpMap[float64] `json:"days"`
		Count     phpMap[int]     `json:"count"`
		ExtCounts phpMap[int]     `json:"ext_counts"`
	}
	v.plain = (*plain)(t)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	t.Days, t.Count, t.ExtCounts = v.Days, v.Count, v.ExtCounts
	return nil
}

// rentDetail adds the parameters every rent read sends: a flat `messages` list and `time_seconds`.
func rentDetail(q url.Values) url.Values {
	q.Set("pagination", "0")
	q.Set("number", "1")
	return q
}

// Get rents a number in country for days days. extension=false makes the rent non-extendable for good
// (the server default, and the usual choice, is true). The price is reserved (zbalance), not charged: the rent
// starts with Status 0 and Checked false and waits up to 20 minutes for its first SMS; the first SMS that is not
// "notify" confirms it (Status 1) and charges the price. Unknown country: *APIError UNDEFINED_COUNTRY.
func (a *RentAPI) Get(ctx context.Context, country, days int, extension bool) (RentItem, error) {
	q := rentDetail(url.Values{"country": {itoa(country)}, "days": {itoa(days)}, "extension": {boolParam(extension)}})
	var res struct {
		Item RentItem `json:"item"`
	}
	err := a.do(ctx, "rent/getRentNum.php", q, nil, &res)
	return res.Item, err
}

// State returns all rents of the account.
func (a *RentAPI) State(ctx context.Context) ([]RentItem, error) {
	var res struct {
		List []RentItem `json:"list"`
	}
	if err := a.do(ctx, "rent/getRentState.php", rentDetail(url.Values{}), nil, &res); err != nil {
		return nil, err
	}
	if res.List == nil {
		return []RentItem{}, nil
	}
	return res.List, nil
}

// StateOne returns one rent, or *APIError ERROR_NO_OPERATIONS when it does not exist.
func (a *RentAPI) StateOne(ctx context.Context, tzid int64) (RentItem, error) {
	var res struct {
		List []RentItem `json:"list"`
	}
	if err := a.do(ctx, "rent/getRentState.php", rentDetail(tzidParam(tzid)), nil, &res); err != nil {
		return RentItem{}, err
	}
	if len(res.List) == 0 {
		return RentItem{}, newAPIError("ERROR_NO_OPERATIONS", "")
	}
	return res.List[0], nil
}

// Extend extends a rent by days days and charges at once (only confirmed rents created with extension=true).
// Every refusal (unknown tzid, unconfirmed rent, not extendable, days not in the tariff, a repeat within 10 s)
// is *APIError TRY_AGAIN_LATER; WARNING_LOW_BALANCE when the balance is short.
func (a *RentAPI) Extend(ctx context.Context, tzid int64, days int) (RentItem, error) {
	q := tzidParam(tzid)
	q.Set("days", itoa(days))
	var res struct {
		Item RentItem `json:"item"`
	}
	err := a.do(ctx, "rent/extendRentState.php", rentDetail(q), nil, &res)
	return res.Item, err
}

// Tariffs returns the rent prices of every country, keyed by country code. No key needed.
func (a *RentAPI) Tariffs(ctx context.Context) (map[string]RentTariff, error) {
	var res phpMap[RentTariff]
	if err := a.do(ctx, "rent/tariffsRent.php", nil, nil, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// TariffsOne returns the rent prices of one country, or *APIError NO_COUNTRY for a country without rent.
// No key needed.
func (a *RentAPI) TariffsOne(ctx context.Context, country int) (RentTariff, error) {
	var raw json.RawMessage
	if err := a.do(ctx, "rent/tariffsRent.php", url.Values{"country": {itoa(country)}}, nil, &raw); err != nil {
		return RentTariff{}, err
	}
	// An unknown country answers `[]`.
	var probe map[string]json.RawMessage
	if t := bytes.TrimSpace(raw); len(t) == 0 || t[0] != '{' || json.Unmarshal(t, &probe) != nil || probe["code"] == nil {
		return RentTariff{}, newAPIError("NO_COUNTRY", "")
	}
	var res RentTariff
	if err := json.Unmarshal(raw, &res); err != nil {
		return RentTariff{}, &TransportError{Err: err, msg: "unexpected response shape: " + err.Error()}
	}
	return res, nil
}

// Close returns the number before the rent period ends. A confirmed rent closes without a refund. An unconfirmed
// one (Status 0) cannot be closed within its first 120 s without any SMS (NO_COMPLETE_TZID); after that, with no
// SMS or only "notify" ones, the reservation is released. Unknown tzid: *APIError ERROR_NO_OPERATIONS.
func (a *RentAPI) Close(ctx context.Context, tzid int64) error {
	return a.do(ctx, "rent/closeRentNum.php", tzidParam(tzid), nil, nil)
}
