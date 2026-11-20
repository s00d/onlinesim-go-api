package onlinesim_test

import (
	"testing"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

func TestFreeListNoSuffixOnlySetParams(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(fixture(t, "getFreeList"))})
	list, err := c.Free().List(ctx, onlinesim.FreeListOptions{Country: 44, Number: "7459608320", Count: 20})
	must(t, err)
	eq(t, f.last().path, "/api/getFreeList")
	eq(t, f.last().query, map[string]string{"country": "44", "number": "7459608320", "count": "20"})
	eq(t, list.Numbers["7459608320"].FullNumber, "447459608320")
	eq(t, list.Messages.Data[0].Code, "195895")
	eq(t, list.Messages.Total, 426)
	eq(t, list.Messages.Number, "7459608320")
	eq(t, list.Countries[1].Country, 380)
}

func TestFreeListAcceptsNullCountryTexts(t *testing.T) {
	c, _ := fakeClient(t, []reply{ok(`{"response":1,"countries":[{"country":999,"country_text":null,"country_original":null}],"numbers":{"1":{"country":999,"country_original":null}},"messages":{"data":[]},"ignore":"<p>x</p>","content":null}`)})
	l, err := c.Free().List(ctx, onlinesim.FreeListOptions{})
	must(t, err)
	eq(t, l.Countries, []onlinesim.FreeCountry{{Country: 999}})
	eq(t, l.Numbers["1"].CountryOriginal, "")
}

func TestFreeListNormalisesEmptyNumbers(t *testing.T) {
	c, f := fakeClient(t, []reply{ok(`{"response":1,"countries":[],"numbers":[],"messages":{"data":[]}}`)})
	list, err := c.Free().List(ctx, onlinesim.FreeListOptions{})
	must(t, err)
	eq(t, list.Numbers, map[string]onlinesim.FreeNumber{})
	eq(t, f.last().query, map[string]string{})
}
