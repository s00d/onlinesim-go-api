package onlinesim

import (
	"context"
	"encoding/json"
	"net/url"
)

// FreeAPI: public free numbers, client.Free(). No key needed.
type FreeAPI struct {
	do doFunc
}

// FreeListOptions selects what List returns. Zero fields are not sent (server defaults).
type FreeListOptions struct {
	// Country code; server default 44.
	Country int
	// Number without the country code, as the keys of FreeList.Numbers.
	Number string
	// Page of Messages.
	Page int
	// Count is the number of SMS per page, 10-500.
	Count int
}

// FreeCountry is a country with free numbers.
type FreeCountry struct {
	Country         int    `json:"country"`
	CountryText     string `json:"country_text"`
	CountryOriginal string `json:"country_original"`
}

// FreeNumber is a free number.
type FreeNumber struct {
	Country         int    `json:"country"`
	CountryOriginal string `json:"country_original"`
	FullNumber      string `json:"full_number"`
	DataHumans      string `json:"data_humans"`
	IsArchive       bool   `json:"is_archive"`
}

// FreeMessage is an SMS of a free number.
type FreeMessage struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
	// InNumber is the sender.
	InNumber string `json:"in_number"`
	MyNumber string `json:"my_number"`
	// Code is "-" when no code was found.
	Code       string `json:"code"`
	CreatedAt  string `json:"created_at"`
	DataHumans string `json:"data_humans"`
}

// FreeMessages is the page of SMS of the selected number.
type FreeMessages struct {
	Paginator[FreeMessage]
	Number  string `json:"number"`
	Country int    `json:"country"`
}

// FreeList is the result of List.
type FreeList struct {
	Countries []FreeCountry `json:"countries"`
	// Numbers is keyed by the number without the country code.
	Numbers  map[string]FreeNumber `json:"numbers"`
	Messages FreeMessages          `json:"messages"`
}

// UnmarshalJSON accepts numbers as `[]`.
func (l *FreeList) UnmarshalJSON(b []byte) error {
	type plain FreeList
	var v struct {
		*plain
		Numbers phpMap[FreeNumber] `json:"numbers"`
	}
	v.plain = (*plain)(l)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	l.Numbers = v.Numbers
	if l.Numbers == nil {
		l.Numbers = map[string]FreeNumber{}
	}
	return nil
}

// List returns the countries, the numbers of the selected country and the SMS of the selected number.
func (a *FreeAPI) List(ctx context.Context, opts FreeListOptions) (FreeList, error) {
	q := url.Values{}
	if opts.Country != 0 {
		q.Set("country", itoa(opts.Country))
	}
	if opts.Number != "" {
		q.Set("number", opts.Number)
	}
	if opts.Page != 0 {
		q.Set("page", itoa(opts.Page))
	}
	if opts.Count != 0 {
		q.Set("count", itoa(opts.Count))
	}
	var res FreeList
	err := a.do(ctx, "getFreeList", q, nil, &res)
	return res, err
}
