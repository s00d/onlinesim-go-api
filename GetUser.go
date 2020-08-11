package onlinesim

import (
	"encoding/json"
	"fmt"
)

type GetUser struct {
	client *Onlinesim
}

type BalanceResponse struct {
	Balance  float64 `json:"balance"`
	Zbalance int     `json:"zbalance"`
	Income   float64 `json:"income"`
}

func (c *GetUser) balance() (error, BalanceResponse) {
	m := make(map[string]string)
	m["income"] = "1"
	result := c.client.get("getBalance", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), BalanceResponse{}
	}

	response := BalanceResponse{}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), BalanceResponse{}
	}

	return nil, response
}

type ProfileResponse struct {
	Response string  `json:"response"`
	Profile  Profile `json:"profile"`
}

type Profile struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	Username      string      `json:"username"`
	Email         string      `json:"email"`
	Apikey        string      `json:"apikey"`
	APIAccess     bool        `json:"api_access"`
	Locale        string      `json:"locale"`
	NumberRegion  interface{} `json:"number_region"`
	NumberCountry string      `json:"number_country"`
	NumberReject  interface{} `json:"number_reject"`
	CreatedAt     string      `json:"created_at"`
	Payment       struct {
		Payment  float64 `json:"payment"`
		Spent    int     `json:"spent"`
		Now      int     `json:"now"`
		Income   float64 `json:"income"`
		SmsCount int     `json:"sms_count"`
	} `json:"payment"`
}

func (c *GetUser) profile() (error, Profile) {
	m := make(map[string]string)
	result := c.client.get("getProfile", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), Profile{}
	}

	response := ProfileResponse{}
	err = json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Profile{}
	}

	return nil, response.Profile
}
