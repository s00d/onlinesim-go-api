package onlinesim

import (
	"context"
	"strconv"
)

// FreeAPI covers free public numbers and messages.
type FreeAPI struct{ c *Client }

// FreeCountry is a country entry from the free list.
type FreeCountry struct {
	Country         int    `json:"country"`
	CountryText     string `json:"country_text"`
	CountryOriginal string `json:"country_original,omitempty"`
}

// FreeNumber is a free phone number.
type FreeNumber struct {
	Maxdate     string `json:"maxdate"`
	Number      string `json:"number"`
	Country     int    `json:"country"`
	UpdatedAt   string `json:"updated_at"`
	DataHumans  string `json:"data_humans"`
	FullNumber  string `json:"full_number"`
	CountryText string `json:"country_text"`
}

// FreeMessage is a free SMS message.
type FreeMessage struct {
	Text       string `json:"text"`
	InNumber   string `json:"in_number"`
	MyNumber   any    `json:"my_number"`
	CreatedAt  string `json:"created_at"`
	DataHumans string `json:"data_humans"`
}

// FreeListNumber is a number entry inside FreeList.
type FreeListNumber struct {
	Country         int    `json:"country"`
	CountryOriginal string `json:"country_original"`
	DataHumans      string `json:"data_humans"`
	FullNumber      string `json:"full_number"`
	IsArchive       bool   `json:"is_archive"`
}

// FreeListMessages is the messages page inside FreeList.
type FreeListMessages struct {
	CurrentPage int           `json:"current_page"`
	From        int           `json:"from"`
	LastPage    int           `json:"last_page"`
	PerPage     int           `json:"per_page"`
	To          int           `json:"to"`
	Total       int           `json:"total"`
	Number      string        `json:"number"`
	Country     int           `json:"country"`
	Data        []FreeMessage `json:"data"`
}

// FreeListResponse is the getFreeList payload.
type FreeListResponse struct {
	Countries []FreeCountry             `json:"countries"`
	Numbers   map[string]FreeListNumber `json:"numbers"`
	Messages  FreeListMessages          `json:"messages"`
}

// Countries returns free countries.
func (a *FreeAPI) Countries(ctx context.Context) ([]FreeCountry, error) {
	var resp struct {
		Response  any           `json:"response"`
		Countries []FreeCountry `json:"countries"`
	}
	err := a.c.tr.GetJSON(ctx, "getFreeCountryList", map[string]string{}, true, &resp)
	return resp.Countries, err
}

// Numbers returns free numbers for a country.
func (a *FreeAPI) Numbers(ctx context.Context, country int) ([]FreeNumber, error) {
	var resp struct {
		Response any          `json:"response"`
		Numbers  []FreeNumber `json:"numbers"`
	}
	err := a.c.tr.GetJSON(ctx, "getFreePhoneList", map[string]string{
		"country": strconv.Itoa(country),
	}, true, &resp)
	return resp.Numbers, err
}

// Messages returns free messages for a phone (page defaults to 1).
func (a *FreeAPI) Messages(ctx context.Context, phone int64, page int) ([]FreeMessage, error) {
	if page <= 0 {
		page = 1
	}
	var resp struct {
		Response any `json:"response"`
		Messages struct {
			Data []FreeMessage `json:"data"`
		} `json:"messages"`
	}
	err := a.c.tr.GetJSON(ctx, "getFreeMessageList", map[string]string{
		"phone": strconv.FormatInt(phone, 10),
		"page":  strconv.Itoa(page),
	}, true, &resp)
	return resp.Messages.Data, err
}

// FreeList returns the combined free list snapshot.
func (a *FreeAPI) FreeList(ctx context.Context) (FreeListResponse, error) {
	var resp FreeListResponse
	err := a.c.tr.GetJSON(ctx, "getFreeList", map[string]string{}, true, &resp)
	return resp, err
}
