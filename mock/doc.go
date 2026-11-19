// Package mock is a local OnlineSim HTTP mock for tests and offline runs: no real numbers,
// no paid calls. It checks paths and the Bearer key like the real API.
//
//	m := mock.Start()
//	defer m.Close()
//	m.ScriptSMS(mock.SmsScript{Service: "telegram", Code: "123456"})
//	client := m.Client()
package mock
