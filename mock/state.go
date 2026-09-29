package mock

import (
	"encoding/json"
	"os"
	"strconv"
)

type operation struct {
	Service         string `json:"service"`
	Number          string `json:"number"`
	Country         int    `json:"country"`
	Code            string `json:"code"`
	Polls           uint   `json:"polls"`
	PollsBeforeCode uint   `json:"polls_before_code"`
	Closed          bool   `json:"closed"`
}

type persistedState struct {
	Version    uint                 `json:"version"`
	Balance    float64              `json:"balance"`
	Zbalance   float64              `json:"zbalance"`
	Income     float64              `json:"income"`
	WebhookURL *string              `json:"webhook_url"`
	NextTzid   int64                `json:"next_tzid"`
	Ops        map[string]operation `json:"ops"`
}

type mockState struct {
	Balance      float64
	Zbalance     float64
	Income       float64
	balanceError *string
	WebhookURL   *string
	NextTzid     int64
	scripts      []SmsScript
	ops          map[int64]operation
	noNumberFor  []string
	rentOps      map[int64]operation
}

func newMockState() *mockState {
	return &mockState{
		Balance:  100,
		Zbalance: 0,
		Income:   0,
		NextTzid: 1000,
		ops:      map[int64]operation{},
		rentOps:  map[int64]operation{},
	}
}

func (s *mockState) toPersisted() persistedState {
	ops := map[string]operation{}
	for id, op := range s.ops {
		ops[strconv.FormatInt(id, 10)] = op
	}
	return persistedState{
		Version:    1,
		Balance:    s.Balance,
		Zbalance:   s.Zbalance,
		Income:     s.Income,
		WebhookURL: s.WebhookURL,
		NextTzid:   s.NextTzid,
		Ops:        ops,
	}
}

func (s *mockState) applyPersisted(p persistedState) {
	s.Balance = p.Balance
	s.Zbalance = p.Zbalance
	s.Income = p.Income
	s.WebhookURL = p.WebhookURL
	if p.NextTzid > 0 {
		s.NextTzid = p.NextTzid
	}
	s.ops = map[int64]operation{}
	for k, op := range p.Ops {
		id, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			continue
		}
		s.ops[id] = op
	}
}

func (m *Server) loadState() error {
	if m.statePath == "" {
		return nil
	}
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var p persistedState
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	m.state.applyPersisted(p)
	return nil
}

func (m *Server) saveLocked() {
	if m.statePath == "" {
		return
	}
	p := m.state.toPersisted()
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m.statePath, data, 0o600)
}
