package onlinesim

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Client-side normalisation of server quirks, shared by the response types and ParseWebhook.

// phpMap decodes a JSON object, and the empty PHP array `[]` (or null) as an empty map.
type phpMap[V any] map[string]V

func (m *phpMap[V]) UnmarshalJSON(b []byte) error {
	out := map[string]V{}
	if t := bytes.TrimSpace(b); len(t) > 0 && t[0] == '{' {
		if err := json.Unmarshal(t, &out); err != nil {
			return err
		}
	}
	*m = out
	return nil
}

// lenientFloat decodes a number or a numeric string ("1442.330"); anything else is 0.
type lenientFloat float64

func (f *lenientFloat) UnmarshalJSON(b []byte) error {
	v, _ := num(b)
	*f = lenientFloat(v)
	return nil
}

// num parses a JSON number or numeric string; ok is false when missing or not numeric.
func num(raw json.RawMessage) (float64, bool) {
	if t := bytes.TrimSpace(raw); len(t) == 0 || string(t) == "null" {
		return 0, false
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return n, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return v, true
		}
	}
	return 0, false
}

// str returns a JSON string as is and a JSON number as its text; "" otherwise.
func str(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}
