package onlinesim

import (
	"encoding/json"
	"fmt"
	"github.com/ddliu/go-httpclient"
	"time"
)

const (
	baseURL   = "https://onlinesim.ru/api"
	rateLimit = 2
)

type Onlinesim struct {
	rateLimiter <-chan time.Time
	baseURL     string
	apiKey      string
	lang        string
	dev_id      string
}

type Default struct {
	Response interface{} `json:"response"`
}

func NewClient(apiKey string, lang string, dev_id string) *Onlinesim {
	if lang == "" {
		lang = "en"
	}

	return &Onlinesim{
		rateLimiter: time.Tick(time.Second / time.Duration(rateLimit)),
		apiKey:      apiKey,
		baseURL:     baseURL,
		lang:        lang,
		dev_id:      dev_id,
	}
}

// SetRateLimit rate limit setter for custom usage
// Onlinesim limit is 5 requests per second (we use 2)
func (at *Onlinesim) SetRateLimit(customRateLimit int) {
	at.rateLimiter = time.Tick(time.Second / time.Duration(customRateLimit))
}

func (at *Onlinesim) rateLimit() {
	<-at.rateLimiter
}

func (at *Onlinesim) get(method string, params map[string]string) []byte {
	at.rateLimit()
	params["apikey"] = at.apiKey
	params["lang"] = at.lang
	params["dev_id"] = at.dev_id

	url := fmt.Sprintf("%s/%s.php", at.baseURL, method)

	httpclient.Defaults(httpclient.Map{
		httpclient.OPT_USERAGENT:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.89 Safari/537.36",
		"Accept-Language":         "en-us",
		httpclient.OPT_PROXY:      "127.0.0.1:4034",
		httpclient.OPT_PROXYTYPE:  httpclient.PROXY_HTTP,
		httpclient.OPT_UNSAFE_TLS: true,
	})

	res, err := httpclient.Get(url, params)
	if err != nil {
		panic(fmt.Errorf("request error: %w", err))
	}
	bodyString, err := res.ToString()
	if err != nil {
		panic(err)
	}

	return []byte(bodyString)
}

func (at *Onlinesim) checkResponse(resp interface{}) error {
	if fmt.Sprintf("%v", resp) != "1" {
		return fmt.Errorf("%s", resp)
	}
	return nil
}

func (at *Onlinesim) checkEmptyResponse(resp []byte) error {
	__default := Default{}
	err := json.Unmarshal(resp, &__default)
	if err == nil {
		if __default.Response == nil {
			return nil
		}
		if fmt.Sprintf("%v", __default.Response) == "" {
			return nil
		}
		if fmt.Sprintf("%v", __default.Response) != "1" {
			return fmt.Errorf("%s", __default.Response)
		}
	}
	return nil
}

func (c *Onlinesim) free() *GetFree {
	return &GetFree{
		client: c,
	}
}

func (c *Onlinesim) numbers() *GetNumbers {
	return &GetNumbers{
		client: c,
	}
}

func (c *Onlinesim) proxy() *GetProxy {
	return &GetProxy{
		client: c,
	}
}

func (c *Onlinesim) rent() *GetRent {
	return &GetRent{
		client: c,
	}
}

func (c *Onlinesim) user() *GetUser {
	return &GetUser{
		client: c,
	}
}
