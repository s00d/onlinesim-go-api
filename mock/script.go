package mock

import "os"

// SmsScript describes one mocked number order and its SMS.
type SmsScript struct {
	Service string
	Number  string
	Country int
	Code    string
	// PollsBeforeCode is the number of getState polls that see no SMS yet.
	PollsBeforeCode uint
	Price           float64
}

// DefaultSmsScript is used for a getNum without a queued script.
func DefaultSmsScript() SmsScript {
	return SmsScript{Service: "telegram", Number: "+19001234567", Country: 1, Code: "123456", Price: 10}
}

// ScriptSMS queues a number and its SMS code for the next getNum of s.Service. Zero fields get
// DefaultSmsScript values.
func (m *Server) ScriptSMS(s SmsScript) {
	d := DefaultSmsScript()
	if s.Service == "" {
		s.Service = d.Service
	}
	if s.Number == "" {
		s.Number = d.Number
	}
	if s.Country == 0 {
		s.Country = d.Country
	}
	if s.Code == "" {
		s.Code = d.Code
	}
	if s.Price == 0 {
		s.Price = d.Price
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.scripts = append(m.state.scripts, s) // scripts are not persisted
}

// FailNoNumber makes getNum for service answer NO_NUMBER.
func (m *Server) FailNoNumber(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.noNumberFor[service] = true
}

// FailBalance makes the next getBalance answer the error code once.
func (m *Server) FailBalance(code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.balanceError = code
}

// SetBalance sets the getBalance values (persisted when a state path is set).
func (m *Server) SetBalance(balance, zbalance, income float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Balance, m.state.Zbalance, m.state.Income = balance, zbalance, income
	m.saveLocked()
}

// AddRentSMS delivers an SMS to a rent created on this mock; false when the tzid is unknown.
func (m *Server) AddRentSMS(tzid int64, service, text, code string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.state.rents[tzid]
	if !ok {
		return false
	}
	sms := map[string]any{"id": len(r.Messages) + 1, "service": service, "text": text, "code": code, "created_at": "2026-01-01 00:00:00"}
	r.Messages = append([]map[string]any{sms}, r.Messages...) // newest first
	return true
}

// StatePathEnv is read by Builder.StatePathFromEnv.
const StatePathEnv = "ONLINESIM_MOCK_STATE"

// ResetState deletes a persisted state file.
func ResetState(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}
