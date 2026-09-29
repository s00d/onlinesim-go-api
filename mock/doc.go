// Package mock provides a local OnlineSim HTTP mock for unit tests and
// offline integration — no real numbers, no paid API calls.
//
//	m := mock.Start()
//	defer m.Close()
//	m.ScriptSMS(mock.SmsScript{Service: "telegram", Code: "123456"})
//	client := m.Client()
package mock
