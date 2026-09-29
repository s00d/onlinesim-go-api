package onlinesim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// RentAPI covers long-term number rent.
type RentAPI struct{ c *Client }

// RentMessage is an SMS inside a rent operation.
type RentMessage struct {
	ID        int64  `json:"id"`
	Service   string `json:"service"`
	Text      string `json:"text"`
	Code      string `json:"code"`
	CreatedAt string `json:"created_at"`
}

// RentItem is an active or newly created rent.
type RentItem struct {
	Tzid      int64         `json:"tzid"`
	Status    int           `json:"status"`
	Messages  []RentMessage `json:"messages"`
	Country   int           `json:"country"`
	Rent      int           `json:"rent"`
	Extension int           `json:"extension"`
	Sum       any           `json:"sum"`
	Number    string        `json:"number"`
	Time      int64         `json:"time"`
	Days      int           `json:"days"`
	Hours     int           `json:"hours"`
	Extend    any           `json:"extend"`
	Checked   bool          `json:"checked"`
	Reload    int           `json:"reload"`
	DayExtend any           `json:"day_extend"`
}

// RentTariff is rent pricing for a country.
type RentTariff struct {
	Code     int                `json:"code"`
	Enabled  bool               `json:"enabled"`
	Name     string             `json:"name"`
	New      bool               `json:"new"`
	Position int                `json:"position"`
	Count    map[string]float64 `json:"count"`
	Days     map[string]float64 `json:"days"`
	Extend   float64            `json:"extend"`
}

// GetRentParams configures a rent order.
type GetRentParams struct {
	Country int
	Days    int
	// Extension: nil defaults to true (Laravel/Rust); use BoolPtr(false) to disable.
	Extension *bool
}

// BoolPtr returns a pointer to v.
func BoolPtr(v bool) *bool { return &v }

// Get orders a rent number.
func (a *RentAPI) Get(ctx context.Context, p GetRentParams) (RentItem, error) {
	if p.Country == 0 {
		p.Country = DefaultCountry
	}
	if p.Days <= 0 {
		p.Days = 1
	}
	ext := true
	if p.Extension != nil {
		ext = *p.Extension
	}
	var resp struct {
		Response any      `json:"response"`
		Item     RentItem `json:"item"`
	}
	err := a.c.tr.GetJSON(ctx, "rent/getRentNum", map[string]string{
		"country":    strconv.Itoa(p.Country),
		"days":       strconv.Itoa(p.Days),
		"extension":  strconv.FormatBool(ext),
		"pagination": "false",
	}, true, &resp)
	return resp.Item, err
}

// State lists active rents.
func (a *RentAPI) State(ctx context.Context) ([]RentItem, error) {
	var resp struct {
		Response any        `json:"response"`
		List     []RentItem `json:"list"`
	}
	err := a.c.tr.GetJSON(ctx, "rent/getRentState", map[string]string{
		"pagination": "false",
	}, true, &resp)
	return resp.List, err
}

// StateOne returns one rent by tzid.
func (a *RentAPI) StateOne(ctx context.Context, tzid int64) (RentItem, error) {
	var resp struct {
		Response any        `json:"response"`
		List     []RentItem `json:"list"`
	}
	err := a.c.tr.GetJSON(ctx, "rent/getRentState", map[string]string{
		"tzid":       strconv.FormatInt(tzid, 10),
		"pagination": "false",
	}, true, &resp)
	if err != nil {
		return RentItem{}, err
	}
	if len(resp.List) == 0 {
		return RentItem{}, fmt.Errorf("no rent for tzid %d", tzid)
	}
	return resp.List[0], nil
}

// Extend extends a rent by days.
func (a *RentAPI) Extend(ctx context.Context, tzid int64, days int) (RentItem, error) {
	if days <= 0 {
		days = 1
	}
	var resp struct {
		Response any      `json:"response"`
		Item     RentItem `json:"item"`
	}
	err := a.c.tr.GetJSON(ctx, "rent/extendRentState", map[string]string{
		"tzid": strconv.FormatInt(tzid, 10),
		"days": strconv.Itoa(days),
	}, true, &resp)
	return resp.Item, err
}

// PortReload reloads the rent port.
func (a *RentAPI) PortReload(ctx context.Context, tzid int64) error {
	return a.c.tr.GetJSON(ctx, "rent/portReload", map[string]string{
		"tzid": strconv.FormatInt(tzid, 10),
	}, true, nil)
}

// Tariffs returns rent tariffs for all countries.
func (a *RentAPI) Tariffs(ctx context.Context) (map[string]RentTariff, error) {
	out := map[string]RentTariff{}
	err := a.c.tr.GetJSON(ctx, "rent/tariffsRent", map[string]string{}, true, &out)
	return out, err
}

// TariffsOne returns rent tariff for one country.
func (a *RentAPI) TariffsOne(ctx context.Context, country int) (RentTariff, error) {
	if country == 0 {
		country = DefaultCountry
	}
	var raw json.RawMessage
	err := a.c.tr.GetJSON(ctx, "rent/tariffsRent", map[string]string{
		"country": strconv.Itoa(country),
	}, true, &raw)
	if err != nil {
		return RentTariff{}, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" || (len(trimmed) > 0 && trimmed[0] == '[') {
		return RentTariff{}, fmt.Errorf("no rent tariff for country %d", country)
	}
	var out RentTariff
	if err := json.Unmarshal(raw, &out); err != nil {
		return RentTariff{}, fmt.Errorf("decode rent tariff: %w", err)
	}
	return out, nil
}

// Close closes a rent.
func (a *RentAPI) Close(ctx context.Context, tzid int64) error {
	return a.c.tr.GetJSON(ctx, "rent/closeRentNum", map[string]string{
		"tzid": strconv.FormatInt(tzid, 10),
	}, true, nil)
}
