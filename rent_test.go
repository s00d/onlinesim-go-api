package onlinesim_test

import (
	"encoding/json"
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func rentItems(t *testing.T) []string {
	t.Helper()
	var raw struct {
		List []json.RawMessage `json:"list"`
	}
	must(t, json.Unmarshal([]byte(fixture(t, "rent_getRentState")), &raw))
	out := make([]string, len(raw.List))
	for i, item := range raw.List {
		out[i] = string(item)
	}
	return out
}

func withDetail(q map[string]string) map[string]string {
	q["pagination"], q["number"] = "0", "1"
	return q
}

func TestRentGetSendsExtensionPaginationNumberAndParsesSum(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"item":` + rentItems(t)[0] + `}`)})
	rent, err := c.Rent().Get(ctx, 7, 3, true)
	must(t, err)
	eq(t, f.last().path, "/api/rent/getRentNum.php")
	eq(t, f.last().query, withDetail(map[string]string{"country": "7", "days": "3", "extension": "1"}))
	eq(t, rent.Sum, 5.14)
	eq(t, rent.Extend, map[string]float64{"1": 14.7, "3": 25.3, "7": 51})
	eq(t, rent.Messages[0].Code, "195895")
	eq(t, rent.TimeSeconds, int64(18034680))
}

// The spec shapes: sum a number (getRentNum) or a decimal string (getRentState), code the integer 0 for a notify SMS.
func TestRentDecodesSumUnionAndNotifyCodeZero(t *testing.T) {
	var item map[string]json.RawMessage
	must(t, json.Unmarshal([]byte(rentItems(t)[0]), &item))
	item["status"], item["checked"], item["sum"] = json.RawMessage(`0`), json.RawMessage(`false`), json.RawMessage(`1.5`)
	item["messages"] = json.RawMessage(`[
		{"id":2,"service":"notify","text":"Messages from this sender are not accepted","code":0,"created_at":"2026-11-19 10:01:12"},
		{"id":1,"service":"Google","text":"G-070051","code":"070051","created_at":"2026-11-19 10:01:00"}]`)
	got, _ := json.Marshal(item)
	item["sum"] = json.RawMessage(`"2.40"`)
	listed, _ := json.Marshal(item)
	c, _ := fakeClient(t, []reply{ok(`{"response":1,"item":` + string(got) + `}`), ok(`{"response":1,"list":[` + string(listed) + `]}`)})
	rent, err := c.Rent().Get(ctx, 7, 3, true)
	must(t, err)
	eq(t, [3]any{rent.Status, rent.Checked, rent.Sum}, [3]any{0, false, 1.5})
	eq(t, []string{rent.Messages[0].Code, rent.Messages[1].Code}, []string{"0", "070051"})
	rent, err = c.Rent().StateOne(ctx, rent.Tzid)
	must(t, err)
	eq(t, rent.Sum, 2.4)
}

func TestRentGetExtensionFalse(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"item":` + rentItems(t)[1] + `}`)})
	_, err := c.Rent().Get(ctx, 7, 1, false)
	must(t, err)
	eq(t, f.last().query["extension"], "0")
}

func TestRentStateNormalisesSumAndEmptyExtend(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "rent_getRentState"))})
	list, err := c.Rent().State(ctx)
	must(t, err)
	eq(t, f.last().path, "/api/rent/getRentState.php")
	eq(t, f.last().query, withDetail(map[string]string{}))
	eq(t, []float64{list[0].Sum, list[1].Sum}, []float64{5.14, 0})
	eq(t, list[1].Extend, map[string]float64{})
	eq(t, list[1].Messages, []onlinesim.RentSMS{})
}

func TestRentStateOne(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"list":[` + rentItems(t)[0] + `]}`)})
	rent, err := c.Rent().StateOne(ctx, 430)
	must(t, err)
	eq(t, rent.Tzid, int64(430))
	eq(t, rent.Sum, 5.14)
	eq(t, f.last().query, withDetail(map[string]string{"tzid": "430"}))
}

func TestRentStateOneUnknown(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":1,"list":[]}`)})
	_, err := c.Rent().StateOne(ctx, 1)
	apiErr(t, err, "ERROR_NO_OPERATIONS")
}

func TestRentExtend(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"item":` + rentItems(t)[0] + `}`)})
	rent, err := c.Rent().Extend(ctx, 430, 3)
	must(t, err)
	eq(t, rent.Tzid, int64(430))
	eq(t, f.last().path, "/api/rent/extendRentState.php")
	eq(t, f.last().query, withDetail(map[string]string{"tzid": "430", "days": "3"}))
}

func TestRentTariffs(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "rent_tariffsRent"))})
	all, err := c.Rent().Tariffs(ctx)
	must(t, err)
	eq(t, f.last().path, "/api/rent/tariffsRent.php")
	eq(t, f.last().query, map[string]string{})
	eq(t, all["7"].Days["7"], 9.11)
	eq(t, all["380"].Enabled, false)
}

func TestRentTariffsOne(t *testing.T) {
	var all map[string]json.RawMessage
	must(t, json.Unmarshal([]byte(fixture(t, "rent_tariffsRent")), &all))
	c, f := fakeClient(t, []reply{ok(string(all["7"]))})
	ru, err := c.Rent().TariffsOne(ctx, 7)
	must(t, err)
	eq(t, f.last().query, map[string]string{"country": "7"})
	eq(t, ru.Code, 7)
	eq(t, ru.Count, map[string]int{"3": 1, "7": 1, "15": 1, "30": 1})
	eq(t, ru.ExtCounts["30"], 1)
}

func TestRentTariffsOneEmptyIsNoCountry(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`[]`)})
	_, err := c.Rent().TariffsOne(ctx, 999)
	apiErr(t, err, "NO_COUNTRY")
}

func TestRentClose(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1"}`), ok(`{"response":"NO_CLOSE_RENT_US"}`)})
	must(t, c.Rent().Close(ctx, 430))
	eq(t, f.paths()[0], "/api/rent/closeRentNum.php")
	apiErr(t, c.Rent().Close(ctx, 1), "NO_CLOSE_RENT_US")
}
