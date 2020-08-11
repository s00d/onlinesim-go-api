package onlinesim

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type GetRent struct {
	client *Onlinesim
}

type Rent struct {
	Status    int `json:"status"`
	Extension int `json:"extension"`
	Messages  []struct {
		ID        int    `json:"id"`
		Service   string `json:"service"`
		Text      string `json:"text"`
		Code      string `json:"code"`
		CreatedAt string `json:"created_at"`
	} `json:"messages"`
	Sum       string        `json:"sum"`
	Country   int           `json:"country"`
	Number    string        `json:"number"`
	Rent      int           `json:"rent"`
	Tzid      int           `json:"tzid"`
	Time      int           `json:"time"`
	Days      int           `json:"days"`
	Hours     int           `json:"hours"`
	Extend    []interface{} `json:"extend"`
	Checked   bool          `json:"checked"`
	Reload    int           `json:"reload"`
	DayExtend int           `json:"day_extend"`
}

type GetRentResponse struct {
	Response interface{} `json:"response"`
	Item     Rent        `json:"item"`
}

func (c *GetRent) get(country int, days int, extension bool) (error, Rent) {
	m := make(map[string]string)
	m["country"] = strconv.Itoa(country)
	m["days"] = strconv.Itoa(days)
	m["extension"] = strconv.FormatBool(extension)
	m["pagination"] = "false"
	result := c.client.get("rent/getRentNum", m)

	response := GetRentResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}
	return nil, response.Item
}

type StateRentResponse struct {
	Response interface{} `json:"response"`
	List     []Rent      `json:"list"`
}

func (c *GetRent) state() (error, []Rent) {
	m := make(map[string]string)
	m["pagination"] = "false"
	result := c.client.get("rent/getRentState", m)

	response := StateRentResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}
	return nil, response.List
}

func (c *GetRent) stateOne(tzid int) (error, Rent) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	m["pagination"] = "false"
	result := c.client.get("rent/getRentState", m)

	response := StateRentResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}
	return nil, response.List[0]
}

func (c *GetRent) extend(tzid int, days int) (error, Rent) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	m["days"] = strconv.Itoa(days)
	result := c.client.get("rent/extendRentState", m)

	response := GetRentResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Rent{}
	}
	return nil, response.Item
}

func (c *GetRent) portReload(tzid int) (error, bool) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	result := c.client.get("rent/portReload", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), false
	}
	return nil, true
}

type TariffsRent struct {
	Code     int            `json:"code"`
	Enabled  bool           `json:"enabled"`
	Name     string         `json:"name"`
	New      bool           `json:"new"`
	Position int            `json:"position"`
	Count    map[string]int `json:"count"`
	Days     map[string]int `json:"days"`
	Extend   int            `json:"extend"`
}

func (c *GetRent) tariffs() (error, map[string]TariffsRent) {
	m := make(map[string]string)
	result := c.client.get("rent/tariffsRent", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	response := map[string]TariffsRent{}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), nil
	}

	return nil, response
}

func (c *GetRent) tariffsOne(country int) (error, TariffsRent) {
	m := make(map[string]string)
	m["country"] = strconv.Itoa(country)
	result := c.client.get("rent/tariffsRent", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), TariffsRent{}
	}

	response := TariffsRent{}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), TariffsRent{}
	}

	return nil, response
}

func (c *GetRent) close(tzid int) (error, bool) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	result := c.client.get("rent/closeRentNum", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), false
	}
	return nil, true
}
