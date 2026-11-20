package onlinesim_test

import (
	"encoding/json"
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func TestPrice(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","price":0.36}`)})
	price, err := c.Numbers().Price(ctx, "vkcom", 44)
	must(t, err)
	eq(t, *price, 0.36)
	eq(t, f.last().path, "/api/getPrice.php")
	eq(t, f.last().query, map[string]string{"service": "vkcom", "country": "44"})
}

func TestPriceNil(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"1","price":null}`)})
	price, err := c.Numbers().Price(ctx, "vkcom", 7)
	must(t, err)
	eq(t, price, (*float64)(nil))
}

func TestGetSendsNoNumberFlagAndRejectAsCommaList(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"tzid":1520}`)})
	tzid, err := c.Numbers().Get(ctx, "google", 44, "4470", "4471")
	must(t, err)
	eq(t, tzid, int64(1520))
	eq(t, f.last().path, "/api/getNum.php")
	eq(t, f.last().query, map[string]string{"service": "google", "country": "44", "reject": "4470,4471"})
}

func TestGetOmitsEmptyReject(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"tzid":1}`)})
	_, err := c.Numbers().Get(ctx, "google", 7)
	must(t, err)
	eq(t, f.last().query, map[string]string{"service": "google", "country": "7"})
}

func TestGetWithNumber(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"tzid":1520,"number":"+447459608320","country":44,"time":1200,"service":"google"}`)})
	order, err := c.Numbers().GetWithNumber(ctx, "google", 44)
	must(t, err)
	eq(t, order, onlinesim.NumberOrder{Tzid: 1520, Number: "+447459608320", Country: 44, Time: 1200, Service: "google"})
	eq(t, f.last().query, map[string]string{"service": "google", "country": "44", "number": "1"})
}

func TestStateEncodingAndParsing(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getState"))})
	list, err := c.Numbers().State(ctx, onlinesim.StateOptions{OrderBy: onlinesim.OrderDesc, MsgList: true})
	must(t, err)
	eq(t, len(list), 2)
	eq(t, list[0].Msg.Text, "195895")
	eq(t, list[0].Response, onlinesim.TZNumAnswer)
	eq(t, list[1].GuardIntervalRemainingSeconds, 120)
	eq(t, list[1].Msg, onlinesim.OperationMsg{})
	eq(t, f.last().path, "/api/getState.php")
	eq(t, f.last().query, map[string]string{"message_to_code": "1", "orderby": "desc", "msg_list": "1", "number": "1"})
}

func TestStateDefaults(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`[]`)})
	_, err := c.Numbers().State(ctx, onlinesim.StateOptions{})
	must(t, err)
	eq(t, f.last().query, map[string]string{"message_to_code": "1", "orderby": "asc", "msg_list": "0", "number": "1"})
}

func TestStateEmptyOnNoOperations(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"ERROR_NO_OPERATIONS"}`)})
	list, err := c.Numbers().State(ctx, onlinesim.StateOptions{})
	must(t, err)
	eq(t, list, []onlinesim.Operation{})
}

func TestStateMsgList(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`[{"tzid":1,"response":"TZ_NUM_ANSWER","msg":[{"service":"vk","msg":"1234"}]},{"tzid":2,"response":"TZ_NUM_WAIT","msg":false}]`)})
	list, err := c.Numbers().State(ctx, onlinesim.StateOptions{MsgList: true})
	must(t, err)
	eq(t, list[0].Msg.List, []onlinesim.OperationSMS{{Service: "vk", Msg: "1234"}})
	eq(t, list[1].Msg, onlinesim.OperationMsg{})
}

func TestStateOne(t *testing.T) {
	var ops []json.RawMessage
	must(t, json.Unmarshal([]byte(fixture(t, "getState")), &ops))
	c, f := fakeClient(t, []reply{ok("[" + string(ops[0]) + "]")})
	op, err := c.Numbers().StateOne(ctx, 594, onlinesim.StateOptions{FullMessage: true})
	must(t, err)
	eq(t, op.Tzid, int64(594))
	eq(t, op.Number, "+77781746533")
	eq(t, op.Sum, 0.3)
	eq(t, f.last().query, map[string]string{"tzid": "594", "message_to_code": "0", "msg_list": "0", "number": "1"})
}

func TestStateOneNoOperations(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"ERROR_NO_OPERATIONS"}`), ok(`[]`)})
	_, err := c.Numbers().StateOne(ctx, 1, onlinesim.StateOptions{})
	apiErr(t, err, "ERROR_NO_OPERATIONS")
	_, err = c.Numbers().StateOne(ctx, 1, onlinesim.StateOptions{})
	apiErr(t, err, "ERROR_NO_OPERATIONS")
}

func TestNext(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"1","tzid":"1520"}`)})
	must(t, c.Numbers().Next(ctx, 1520))
	eq(t, f.last().path, "/api/setOperationRevise.php")
	eq(t, f.last().query, map[string]string{"tzid": "1520"})
}

func TestClose(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"tzid":1520}`)})
	must(t, c.Numbers().Close(ctx, 1520))
	eq(t, f.last().path, "/api/setOperationOk.php")
	eq(t, f.last().query, map[string]string{"tzid": "1520"})
}

func TestBan(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"tzid":1520}`)})
	must(t, c.Numbers().Ban(ctx, 1520))
	eq(t, f.last().query, map[string]string{"tzid": "1520", "ban": "1"})
}

// The client never retries: the server locks the tzid for 5 s and the caller decides.
func TestCloseSurfacesErrorsWithoutRetry(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":"NO_COMPLETE_TZID"}`), ok(`{"response":"INTERVAL_CONCURRENT_REQUESTS_ERROR"}`)})
	apiErr(t, c.Numbers().Close(ctx, 1), "NO_COMPLETE_TZID")
	apiErr(t, c.Numbers().Close(ctx, 1), "INTERVAL_CONCURRENT_REQUESTS_ERROR")
	eq(t, len(f.paths()), 2)
}

func TestBanAfterSMSIsAPIErrorZero(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"0"}`)})
	err := c.Numbers().Ban(ctx, 1520)
	eq(t, apiErr(t, err, "0").Message, "Ban not applied: the operation already has an SMS")
}

func TestNextAcceptsStringTzidAnswerAndWrongTzid(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"tzid":"1520","response":"1"}`), ok(`{"response":"ERROR_WRONG_TZID"}`)})
	must(t, c.Numbers().Next(ctx, 1520))
	apiErr(t, c.Numbers().Next(ctx, 1), "ERROR_WRONG_TZID")
}

func TestStateOneWithoutResponse(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`[{"tzid":5,"country":7,"service":"vkcom","sum":0.5,"time":900,"form":"index"}]`)})
	op, err := c.Numbers().StateOne(ctx, 5, onlinesim.StateOptions{})
	must(t, err)
	eq(t, op.Tzid, int64(5))
	eq(t, op.Response, onlinesim.OperationStatus(""))
}

func TestTariffsCountryAllAndEmptyServices(t *testing.T) {
	uk := fixture(t, "getNumbersStats_44")
	c, f := fakeClient(t, []reply{ok(`{"44":` + uk + `,"7":{"name":"russia","position":1,"code":7,"other":false,"new":false,"enabled":true,"services":[]}}`)})
	all, err := c.Numbers().Tariffs(ctx)
	must(t, err)
	eq(t, f.last().path, "/api/getNumbersStats.php")
	eq(t, f.last().query, map[string]string{"country": "all"})
	eq(t, all["44"].Services["service_vkcom"].Slug, "vkcom")
	eq(t, all["7"].Services, map[string]onlinesim.ServiceStat{})
	eq(t, all["7"].Other, 0.0)
}

func TestTariffsOne(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getNumbersStats_44"))})
	uk, err := c.Numbers().TariffsOne(ctx, 44)
	must(t, err)
	eq(t, f.last().query, map[string]string{"country": "44"})
	eq(t, uk.Code, 44)
	eq(t, uk.LocaleName, "United Kingdom")
	eq(t, uk.Other, 0.63)
	eq(t, uk.Services["service_vkcom"], onlinesim.ServiceStat{Count: 363, Code: 44, Price: 0.36, ID: 1, Service: "VK", Slug: "vkcom"})
}

func TestServiceStatLenientCountAndPrice(t *testing.T) {
	var s onlinesim.ServiceStat
	must(t, json.Unmarshal([]byte(`{"count":"?","price":"12.50","slug":"x"}`), &s))
	eq(t, s, onlinesim.ServiceStat{Count: -1, Price: 12.5, Slug: "x"})
}

func TestTariffsOneNoCountry(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":"NO_COUNTRY"}`)})
	_, err := c.Numbers().TariffsOne(ctx, 999)
	apiErr(t, err, "NO_COUNTRY")
}
