package auth

import (
	"errors"
	"meguca/config"
	"sync"
	"time"
)

// Score values of various actions
const (
	// Single character update
	CharScore = time.Minute / 350

	// Post creation
	PostCreationScore = time.Second * 10

	// Image insertion score
	ImageScore = time.Second * 20
)

var (
	// The poster is almost certainly spamming
	ErrSpamDected = errors.New("spam detected")

	spamCounters = spamCounterMap{
		m: make(map[string]*spamCounter, 64),
	}
)

type spamCounterMap struct {
	sync.Mutex
	m map[string]*spamCounter
}

func (m *spamCounterMap) get(ip string) *spamCounter {
	m.Lock()
	defer m.Unlock()

	s := m.m[ip]
	if s == nil {
		s = &spamCounter{
			ip: ip,
		}
		s.init()
		m.m[ip] = s
	}
	return s
}

func (m *spamCounterMap) clear() {
	m.Lock()
	defer m.Unlock()
	m.m = make(map[string]*spamCounter, 64)
}

// Deletes expired counters
func (m *spamCounterMap) deleteExpired() {
	m.Lock()
	defer m.Unlock()

	now := time.Now()
	for ip, s := range m.m {
		s.RLock()
		if now.Sub(s.counter) > time.Minute*15 {
			delete(m.m, ip)
		}
		s.RUnlock()
	}
}

func init() {
	go func() {
		t := time.Tick(time.Minute * 10)
		for {
			<-t
			spamCounters.deleteExpired()
		}
	}()
}

type spamCounter struct {
	sync.RWMutex
	ip      string
	counter time.Time
}

// Set the counter to an initial position
func (s *spamCounter) init() {
	s.counter = time.Now().Add(-time.Minute)
}

// Can this IP create a new post?
func (s *spamCounter) canPost() bool {
	s.RLock()
	defer s.RUnlock()
	return s.counter.Before(time.Now())
}

// Increment spam detection score, after performing an action.
// Returns, if the limit was exceeded.
func (s *spamCounter) increment(by time.Duration) (bool, error) {
	now := time.Now()
	s.Lock()
	defer s.Unlock()

	// Keep score from desending bellow offset or initialize
	if now.Sub(s.counter) > time.Minute {
		s.init()
	}
	s.counter = s.counter.Add(by)

	if s.counter.Sub(now) > time.Minute*10 {
		// This surely is not done by normal human interaction
		return true, ErrSpamDected
	}
	return s.counter.After(now), nil
}

func (s *spamCounter) reset() {
	s.Lock()
	defer s.Unlock()
	s.init()
}

// Returns, if the user does not trigger antispam
func CanPost(ip string) bool {
	if !config.Get().Captcha {
		return true
	}
	return spamCounters.get(ip).canPost()
}

// Increment spam detection score to an IP, after performing an action.
// Returns, if the limit was exceeded.
func IncrementSpamScore(ip string, score time.Duration) (bool, error) {
	if !config.Get().Captcha {
		return false, nil
	}
	return spamCounters.get(ip).increment(score)
}

// Reset a spam score to zero by IP
func ResetSpamScore(ip string) {
	if !config.Get().Captcha {
		return
	}
	spamCounters.get(ip).reset()
}

// Clear all spam detection data. Only use for tests.
func ClearSpamCounters() {
	spamCounters.clear()
}
