package onlinesim

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type GetProxy struct {
	client *Onlinesim
}

type ProxyResponse struct {
	Response string `json:"response"`
	Item     Proxy  `json:"item"`
}

type Proxy struct {
	Type           string      `json:"type"`
	ConnectType    string      `json:"connect_type"`
	Host           string      `json:"host"`
	Port           int         `json:"port"`
	User           string      `json:"user"`
	Pass           string      `json:"pass"`
	Operator       string      `json:"operator"`
	Rent           interface{} `json:"rent"`
	GeneralTraffic int         `json:"general_traffic"`
	Traffic        int         `json:"traffic"`
	Country        string      `json:"country"`
	City           string      `json:"city"`
	Session        bool        `json:"session"`
	PortCount      int         `json:"port_count"`
	Rotate         interface{} `json:"rotate"`
	StopAt         string      `json:"stop_at"`
	UpdatedAt      string      `json:"updated_at"`
	CreatedAt      string      `json:"created_at"`
	Tzid           int         `json:"tzid"`
	Time           int         `json:"time"`
	Days           int         `json:"days"`
	Hours          int         `json:"hours"`
	ChangeIP       bool        `json:"change_ip"`
	ChangeType     bool        `json:"change_type"`
}

func (c *GetProxy) GetDays(proxy_type string) (error, Proxy) {
	m := make(map[string]string)
	m["type"] = proxy_type
	m["class"] = "days"
	result := c.client.get("proxy/getProxy", m)

	response := ProxyResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}
	return nil, response.Item
}

func (c *GetProxy) GetTraffic(traffic string) (error, Proxy) {
	m := make(map[string]string)
	m["class"] = "traffic"
	m["count"] = traffic
	result := c.client.get("proxy/getProxy", m)

	response := ProxyResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}
	return nil, response.Item
}

type ProxyStateResponse struct {
	Response interface{} `json:"response"`
	List     []Proxy     `json:"list"`
}

func (c *GetProxy) State(orderby string) (error, []Proxy) {
	m := make(map[string]string)
	m["orderby"] = orderby
	result := c.client.get("proxy/getState", m)

	response := ProxyStateResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), []Proxy{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), []Proxy{}
	}
	return nil, response.List
}

func (c *GetProxy) StateOne(tzid int) (error, Proxy) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	result := c.client.get("proxy/getState", m)

	response := ProxyStateResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), Proxy{}
	}
	return nil, response.List[0]
}

func (c *GetProxy) ChangeIp(tzid int) (error, bool) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	result := c.client.get("proxy/changeIp", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), false
	}

	return nil, true
}

type ChangeTypeResponse struct {
	Response    interface{} `json:"response"`
	ConnectType string      `json:"connect_type"`
}

func (c *GetProxy) ChangeType(tzid int) (error, string) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	result := c.client.get("proxy/changeType", m)

	response := ChangeTypeResponse{}
	err := json.Unmarshal(result, &response)
	if err != nil {
		return fmt.Errorf("%w", err), ""
	}

	err = c.client.checkResponse(response.Response)
	if err != nil {
		return fmt.Errorf("%w", err), ""
	}

	return nil, response.ConnectType
}

func (c *GetProxy) SetComment(tzid int, comment string) (error, bool) {
	m := make(map[string]string)
	m["tzid"] = strconv.Itoa(tzid)
	m["comment"] = comment
	result := c.client.get("proxy/setComment", m)

	err := c.client.checkEmptyResponse(result)
	if err != nil {
		return fmt.Errorf("%w", err), false
	}

	return nil, true
}
