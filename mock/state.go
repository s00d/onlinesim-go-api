package mock

import (
	"math"
	"sort"
	"strconv"
	"time"
	"unicode"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
)

// state is everything the fake backend remembers: balance, operations, rents, the profile's webhook URL
// and its delivery log, scheduled SMS, the failure queue and the request log. Mock owns it and touches it
// only under its lock; the handlers (handlers.go) get it from there.
//
// Deterministic: ids and numbers come from a counter, timestamps from a fixed epoch plus that counter.
// Money is kept in integer thousandths of a USD ("mills") so sums never drift.
type state struct {
	apiKey        string
	guardInterval int
	prices        Prices
	rentTariffs   map[int]RentTariff
	startBalance  int64

	balance     int64 // total incl. the frozen part, mills
	frozen      int64 // frozen by open operations and unconfirmed rents, mills
	operations  map[int64]*Operation
	rents       map[int64]*Rent
	webhookURL  *string
	webhookLogs []webhookLog
	requests    []Request
	failures    []failure
	scheduled   map[int64][]scheduledSMS
	seq         int64 // source of tzids, numbers, message ids and timestamps
}

// Prices of numbers: country code → service slug → USD.
type Prices map[int]map[string]float64

// RentTariff is the rent tariff of one country: rent days → USD, and whether rents can be extended (1) or not (0).
type RentTariff struct {
	Days   map[int]float64
	Extend int
}

// SMS is one SMS received by an operation or a rent.
type SMS struct {
	Service string
	Text    string
	// Code extracted from Text (the first run of 4–8 digits), "" when there is none.
	Code string
}

// Operation is a number operation as the mock stores it.
type Operation struct {
	Tzid    int64
	Service string
	Country int
	// Number is the full number with "+".
	Number string
	// Sum is the price, USD. Frozen while open; spent on close after an SMS, refunded on close without one.
	Sum float64
	// Status is TZ_NUM_WAIT, TZ_NUM_ANSWER or TZ_OVER_OK.
	Status onlinesim.OperationStatus
	// Messages, oldest first.
	Messages []SMS
	// Cursor is the index of the SMS shown in `msg`; Next (setOperationRevise) moves it past the last one.
	Cursor int
}

// RentSMS is one SMS of a rent.
type RentSMS struct {
	SMS
	ID        int64
	CreatedAt string
}

// Rent is a rent as the mock stores it. Closed rents are removed.
type Rent struct {
	Tzid    int64
	Country int
	// Number without the country code, as the backend sends it for rents.
	Number string
	// Days is the total rent days: the initial period plus extensions.
	Days int
	// InitialDays is the initial period (days of getRentNum).
	InitialDays int
	// Status is 0 while the rent is reserved and waits for its first SMS, 1 once an SMS that is not "notify"
	// confirmed it and the price was charged.
	Status int
	// Sum is the total cost incl. extensions, USD. Frozen while Status is 0.
	Sum       float64
	Extension bool
	// Messages, oldest first.
	Messages []RentSMS
}

// Request is one request the mock received.
type Request struct {
	Method string
	// Path relative to `/api/`, e.g. "getNum.php", "rent/getRentNum.php", "getFreeList".
	Path  string
	Query map[string]string
	// Body is the decoded JSON body of a POST, nil for GET.
	Body any
}

type webhookLog struct {
	ID         int64          `json:"id"`
	UserID     int64          `json:"user_id"`
	WebhookURL string         `json:"webhook_url"`
	Type       string         `json:"type"`
	Params     map[string]any `json:"params"`
	Status     string         `json:"status"`
	Error      *string        `json:"error"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

// failure is a queued failure for the next request (to endpoint only, when set): `{"response": code}`,
// or a non-JSON body when code is "", with HTTP status.
type failure struct {
	endpoint string
	code     string
	status   int
}

type scheduledSMS struct {
	text    string
	service string
	polls   int // state polls of this tzid still to pass before the SMS arrives
}

const (
	userID = 1
	// operationTime is the seconds an operation stays open, as reported in `time`. No clock: it never runs down.
	operationTime = 1200
	// rentFirstSMSMinutes is how long an unconfirmed rent waits for its first SMS (stop_time = start + this).
	rentFirstSMSMinutes = 20
	// notify is the sender of a rent SMS from a prohibited sender: it gets the integer code 0 and confirms nothing.
	notify = "notify"
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// stamp is `YYYY-MM-DD HH:MM:SS` at seconds after the fixed mock epoch.
func stamp(seconds int64) string {
	return epoch.Add(time.Duration(seconds) * time.Second).Format(time.DateTime)
}

func toMills(usd float64) int64     { return int64(math.Round(usd * 1000)) }
func fromMills(mills int64) float64 { return float64(mills) / 1000 }

// shortNumber is the deterministic 9-digit (country-less) number of an id.
func shortNumber(id int64) string { return strconv.FormatInt(900_000_000+id, 10) }

func defaultPrices() Prices {
	return Prices{
		7:  {"vkcom": 0.5, "telegram": 1.2, "google": 0.3},
		44: {"google": 0.2, "vkcom": 0.36},
	}
}

func defaultRentTariffs() map[int]RentTariff {
	return map[int]RentTariff{7: {Days: map[int]float64{1: 1.0, 3: 2.5, 7: 5.0}, Extend: 1}}
}

// extractCode is the code rule shared by every SDK mock: dashes/spaces between digits are removed, then the
// first run of 4–8 digits (not part of a longer run) is the code; "" when there is none.
func extractCode(text string) string {
	rs := []rune(text)
	isDigit := func(r rune) bool { return r >= '0' && r <= '9' }
	isSep := func(r rune) bool { return r == '-' || unicode.IsSpace(r) }
	joined := make([]rune, 0, len(rs))
	for i := 0; i < len(rs); i++ {
		if isSep(rs[i]) && i > 0 && isDigit(rs[i-1]) {
			j := i
			for j < len(rs) && isSep(rs[j]) {
				j++
			}
			if j < len(rs) && isDigit(rs[j]) {
				i = j - 1
				continue
			}
		}
		joined = append(joined, rs[i])
	}
	for i := 0; i < len(joined); {
		if !isDigit(joined[i]) {
			i++
			continue
		}
		j := i
		for j < len(joined) && isDigit(joined[j]) {
			j++
		}
		if n := j - i; n >= 4 && n <= 8 {
			return string(joined[i:j])
		}
		i = j
	}
	return ""
}

func newState(o options) *state {
	s := &state{
		apiKey:        o.apiKey,
		guardInterval: o.guardInterval,
		prices:        o.prices,
		rentTariffs:   o.rentTariffs,
		startBalance:  toMills(o.balance),
	}
	s.reset()
	return s
}

func (s *state) reset() {
	s.balance = s.startBalance
	s.frozen = 0
	s.operations = map[int64]*Operation{}
	s.rents = map[int64]*Rent{}
	s.webhookURL = nil
	s.webhookLogs = nil
	s.requests = nil
	s.failures = nil
	s.scheduled = map[int64][]scheduledSMS{}
	s.seq = 0
}

// available is the balance minus the frozen part, mills.
func (s *state) available() int64 { return s.balance - s.frozen }

func (s *state) nextID() int64 {
	s.seq++
	return s.seq
}

// takeFailure removes and returns the first queued failure that applies to path.
func (s *state) takeFailure(path string) (failure, bool) {
	for i, f := range s.failures {
		if f.endpoint == "" || f.endpoint == path {
			s.failures = append(s.failures[:i:i], s.failures[i+1:]...)
			return f, true
		}
	}
	return failure{}, false
}

// deliver makes an SMS arrive at operation or rent tzid now; false when there is no open one.
// An empty service means the operation's service ("other" for a rent).
func (s *state) deliver(tzid int64, text, service string) bool {
	sms := SMS{Service: service, Text: text, Code: extractCode(text)}
	if op, ok := s.operations[tzid]; ok && op.Status != onlinesim.TZOverOK {
		if sms.Service == "" {
			sms.Service = op.Service
		}
		op.Messages = append(op.Messages, sms)
		op.Status = onlinesim.TZNumAnswer
		// Activation: "+<digits>", seconds left, the order time.
		s.webhook("receiving_sms", tzid, op.Country, op.Number, sms, operationTime, stamp(op.Tzid))
		return true
	}
	if rent, ok := s.rents[tzid]; ok {
		if sms.Service == "" {
			sms.Service = "other"
		}
		id := s.nextID()
		rent.Messages = append(rent.Messages, RentSMS{SMS: sms, ID: id, CreatedAt: stamp(id)})
		if rent.Status == 0 && sms.Service != notify {
			// The first real SMS confirms the rent: the reservation is charged.
			cost := toMills(rent.Sum)
			s.frozen -= cost
			s.balance -= cost
			rent.Status = 1
		}
		// Rent: the full number in digits without "+", minutes left, the SMS time.
		s.webhook("rent_sms", tzid, rent.Country, strconv.Itoa(rent.Country)+rent.Number, sms, rent.minutesLeft(), stamp(id))
		return true
	}
	return false
}

func (s *state) schedule(tzid int64, text string, polls int, service string) {
	s.scheduled[tzid] = append(s.scheduled[tzid], scheduledSMS{text: text, service: service, polls: polls})
}

// poll is one state poll of tzid: delivers the scheduled SMS whose polls ran out, counts down the rest.
func (s *state) poll(tzid int64) {
	list, ok := s.scheduled[tzid]
	if !ok {
		return
	}
	var due, rest []scheduledSMS
	for _, sms := range list {
		if sms.polls <= 0 {
			due = append(due, sms)
		} else {
			sms.polls--
			rest = append(rest, sms)
		}
	}
	if len(rest) > 0 {
		s.scheduled[tzid] = rest
	} else {
		delete(s.scheduled, tzid)
	}
	// Dropped silently when the operation was closed meanwhile: a real SMS would not arrive either.
	for _, sms := range due {
		s.deliver(tzid, sms.text, sms.service)
	}
}

// minutesLeft is the rent time in minutes: the whole period once confirmed, the first-SMS window before.
// No clock: it never runs down.
func (r *Rent) minutesLeft() int {
	if r.Status == 1 {
		return r.Days * 24 * 60
	}
	return rentFirstSMSMinutes
}

// webhook records what the backend would have sent; the mock makes no HTTP call.
func (s *state) webhook(kind string, tzid int64, country int, number string, sms SMS, timeLeft int, timeStart string) {
	if s.webhookURL == nil {
		return
	}
	t := stamp(s.seq)
	s.webhookLogs = append(s.webhookLogs, webhookLog{
		ID: int64(len(s.webhookLogs) + 1), UserID: userID, WebhookURL: *s.webhookURL, Type: kind,
		Params: map[string]any{
			"webhook_type": kind, "operation_id": tzid, "user_id": userID, "country_code": country,
			"number": number, "sender": sms.Service, "message": sms.Text, "code": sms.Code,
			"time_start": timeStart, "time_left": timeLeft,
		},
		Status: "success", CreatedAt: t, UpdatedAt: t,
	})
}

// sortedOperations returns the operations by tzid, oldest first.
func (s *state) sortedOperations() []*Operation {
	out := make([]*Operation, 0, len(s.operations))
	for _, op := range s.operations {
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tzid < out[j].Tzid })
	return out
}

// sortedRents returns the rents by tzid, oldest first.
func (s *state) sortedRents() []*Rent {
	out := make([]*Rent, 0, len(s.rents))
	for _, r := range s.rents {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tzid < out[j].Tzid })
	return out
}
