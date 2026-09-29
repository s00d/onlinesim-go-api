package mock

import "os"

// SmsScript describes one mocked SMS purchase + delivery.
type SmsScript struct {
	Service          string
	Number           string
	Country          int
	Code             string
	PollsBeforeCode  uint
	Price            string
}

// DefaultSmsScript returns a telegram script with code 123456.
func DefaultSmsScript() SmsScript {
	return SmsScript{
		Service: "telegram",
		Number:  "+19001234567",
		Country: 7,
		Code:    "123456",
		Price:   "10",
	}
}

// ScriptSMS queues a number + SMS code delivery for the next matching getNum.
func (m *Server) ScriptSMS(s SmsScript) {
	if s.Service == "" {
		s.Service = "telegram"
	}
	if s.Number == "" {
		s.Number = "+19001234567"
	}
	if s.Country == 0 {
		s.Country = 7
	}
	if s.Code == "" {
		s.Code = "123456"
	}
	if s.Price == "" {
		s.Price = "10"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.scripts = append(m.state.scripts, s)
	// scripts are ephemeral — do not persist
}

// FailNoNumber forces getNum for service to return NO_NUMBER.
func (m *Server) FailNoNumber(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.noNumberFor = append(m.state.noNumberFor, service)
}

// FailBalance makes the next getBalance return an API error once.
func (m *Server) FailBalance(code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.balanceError = &code
}

// SetBalance controls getBalance values (persisted when state path is set).
func (m *Server) SetBalance(balance, zbalance, income float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.Balance = balance
	m.state.Zbalance = zbalance
	m.state.Income = income
	m.saveLocked()
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
