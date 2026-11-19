// Package mock is a fake OnlineSim API for testing integrations offline: no network, no key, no money,
// no clock. It answers every endpoint the SDK calls like the real backend (same shapes, same error codes,
// HTTP 456 for a wrong key) and keeps a balance, operations and rents. Deterministic: ids and numbers come
// from a counter.
//
// In-process (no listener): Mock is an http.RoundTripper.
//
//	m := mock.New(mock.WithBalance(10))
//	client := m.Client() // a normal *onlinesim.Client wired to m
//
// Over HTTP, for code that needs a URL: Start serves the same mock on 127.0.0.1.
//
//	srv := mock.Start()
//	defer srv.Close()
//	client := onlinesim.New(mock.DefaultAPIKey, onlinesim.WithBaseURL(srv.BaseURL()))
//
// Test helpers: AddSMS, AddSMSAfter, FailNext, FailNextTransport, SetBalance, Balance, Operations, Rents,
// Requests, Reset.
package mock
