package onlinesim

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type CountriesResponse struct {
	Response  string    `json:"response"`
	Countries []Country `json:"countries"`
}

type Country struct {
	Country     int    `json:"country"`
	CountryText string `json:"country_text"`
}

type NumbersResponse struct {
	Response string   `json:"response"`
	Numbers  []Number `json:"numbers"`
}

type Number struct {
	Maxdate     string `json:"maxdate"`
	Number      string `json:"number"`
	Country     int    `json:"country"`
	UpdatedAt   string `json:"updated_at"`
	DataHumans  string `json:"data_humans"`
	FullNumber  string `json:"full_number"`
	CountryText string `json:"country_text"`
}

type MessagesResponse struct {
	Response string    `json:"response"`
	Numbers  []Message `json:"numbers"`
}

type Message struct {
	Maxdate     string `json:"maxdate"`
	Number      string `json:"number"`
	Country     int    `json:"country"`
	UpdatedAt   string `json:"updated_at"`
	DataHumans  string `json:"data_humans"`
	FullNumber  string `json:"full_number"`
	CountryText string `json:"country_text"`
}

type GetFree struct {
	client *Onlinesim
}

func (c *GetFree) Countries() (error, []Country) {
	m := make(map[string]string)
	result := c.client.get("getFreeCountryList", m)

	response := CountriesResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	return nil, response.Countries
}

func (c *GetFree) Numbers(country int) (error, []Number) {
	m := make(map[string]string)
	m["country"] = strconv.Itoa(country)
	result := c.client.get("getFreePhoneList", m)

	response := NumbersResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	return nil, response.Numbers
}

//
func (c *GetFree) Messages(phone, page int) (error, []Message) {
	if page == 0 {
		page = 1
	}
	m := make(map[string]string)
	m["phone"] = strconv.Itoa(phone)
	m["page"] = strconv.Itoa(page)
	result := c.client.get("getFreePhoneList", m)

	response := MessagesResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	return nil, response.Numbers
}
