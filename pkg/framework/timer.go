package framework

import (
	"net/http"
	"strconv"
	"time"
)

type timerServer struct {
	min map[string]*time.Time
	max map[string]*time.Time
}

func newTimerServer() *timerServer {
	return &timerServer{
		min: make(map[string]*time.Time),
		max: make(map[string]*time.Time),
	}
}

const maxTimeDelta = 300 * time.Millisecond

func (s *timerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	ts := r.URL.Query().Get("time")

	if name == "" || ts == "" {
		http.Error(w, "name and time must be provided", http.StatusBadRequest)
		return
	}

	tI64, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := time.Unix(0, tI64)

	if time.Now().Sub(t).Abs() > maxTimeDelta {
		http.Error(w, "time is too far in the past or future", http.StatusBadRequest)
	}

	s.saveTime(name, &t)
	w.WriteHeader(http.StatusOK)
}

func (s *timerServer) saveTime(name string, t *time.Time) {
	if min, ok := s.min[name]; !ok || min == nil || t.Before(*s.min[name]) {
		s.min[name] = t
	}

	if max, ok := s.max[name]; !ok || max == nil || t.After(*s.max[name]) {
		s.max[name] = t
	}
}

func (s *timerServer) Get(name string) (time.Duration, bool) {
	min, ok := s.min[name]
	if !ok || min == nil {
		return 0, false
	}

	max, ok := s.max[name]
	if !ok || max == nil {
		return 0, false
	}

	return max.Sub(*min), true
}

func (s *timerServer) GetMin(name string) (time.Time, bool) {
	min, ok := s.min[name]
	if !ok || min == nil {
		return time.Time{}, false
	}

	return *min, true
}

func (s *timerServer) GetMax(name string) (time.Time, bool) {
	max, ok := s.max[name]
	if !ok || max == nil {
		return time.Time{}, false
	}

	return *max, true
}

var timerServerInstance *timerServer = nil

func TimerServer() *timerServer {
	if timerServerInstance == nil {
		timerServerInstance = newTimerServer()
	}

	return timerServerInstance
}

const timerServerPort = ":23456"

func StartTimerServer() {
	NilOrPanic(http.ListenAndServe(":23456", TimerServer()))
}
