package websockets

import (
	"encoding/json"
	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	"sync"
	"time"
)

// Subs is the only instance of SubscriptionMap in this running instance, that
// constains and manages all active subscriptions
var Subs = SubscriptionMap{
	subs: make(map[int64]*Subscription),
}

// SubscriptionMap contains all active Subscriptions to threads and boards
type SubscriptionMap struct {
	sync.RWMutex
	subs map[int64]*Subscription
}

// ListenTo assigns a client to listen to the specified subscription. If the
// subscription is not currently active, it is created.
func (s *SubscriptionMap) ListenTo(id int64, client *Client) {
	s.Lock()
	defer s.Unlock()
	var sub *Subscription
	if existing, ok := s.subs[id]; ok {
		sub = existing
	} else {
		created, err := s.newSubsctiption(id)
		if err != nil {
			client.logError(util.WrapError("Error assigning subscription", err))
			return
		}
		sub = created
	}
	sub.Add(client.sender, client.ID)
}

// newSubsctiption creates and initializes a new Subscription instance
func (s *SubscriptionMap) newSubsctiption(id int64) (*Subscription, error) {
	sub := &Subscription{
		id:      id,
		clients: make(subscribedCleints),
		add:     make(chan addRequest),
		remove:  make(chan string),
		write:   make(chan []byte),
		getJSON: make(chan chan []byte),
		close:   make(chan error),
	}
	if err := sub.Open(); err != nil {
		return nil, err
	}
	s.subs[id] = sub
	return sub, nil
}

// Unlisten removes a listener from a subscription and removes the subscription,
// if it no longer has any listeners.
func (s *SubscriptionMap) Unlisten(subID int64, clientID string) {
	s.Lock()
	defer s.Unlock()
	sub, ok := s.subs[subID]
	if ok {
		sub.Remove(clientID)
	}
}

// Exists returns weather a subscription for a thread already exists
func (s *SubscriptionMap) Exists(id int64) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.subs[id]
	return ok
}

// Remove a subscription from the subscription map
func (s *SubscriptionMap) Remove(id int64) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs, id)
}

// ThreadJSON fetches the post data of a thread, if it exists. Otherwise
// returns a nil slice. This used to insure consistency, by making it impossible
// for writes to the thread to happen during the fetch.
func (s *SubscriptionMap) ThreadJSON(id int64) []byte {
	s.RLock()
	defer s.RUnlock()
	sub, ok := s.subs[id]
	if !ok {
		return nil
	}
	return sub.GetJSON()
}

// Subscription manages a map of listener `chan []byte` and sends events to all
// of them, allowing for thread-safe eventful distribution
type Subscription struct {
	id      int64
	counter int64
	add     chan addRequest
	remove  chan string
	write   chan []byte
	getJSON chan chan []byte
	close   chan error
	clients subscribedCleints
	data    struct {
		counter int64
		buffer  []byte
	}
	log []string
}

// Request for adding a Client to a Subscription
type addRequest struct {
	id     string
	client chan<- []byte
}

type subscribedCleints map[string]chan<- []byte

// Open intializes the Subscription and start it's internal loop
func (s *Subscription) Open() error {
	log, err := db.ReplicationLog(s.id)
	if err != nil {
		return util.WrapError("Error opening subscription", err)
	}
	s.counter = int64(len(log))
	s.log = log
	go s.loop()
	return nil
}

// Add adds a Client to listen on the Subscription
func (s *Subscription) Add(client chan<- []byte, id string) {
	s.add <- addRequest{
		id:     id,
		client: client,
	}
}

// Remove removes a client from the Subscription
func (s *Subscription) Remove(id string) {
	s.remove <- id
}

// GetJSON retrieves cached thread post data or reads new data from the
// database, if the cached data is more than 100 messages outdated.
func (s *Subscription) GetJSON() []byte {
	req := make(chan []byte)
	s.getJSON <- req
	return <-req
}

// loop handles the internal channel messages
func (s *Subscription) loop() {
	defer func() { // Remove Subscription, when loop stops
		// TODO: Some kind of client redirecting logic. Too early to
		// implement right now.

		s.clients = nil
		Subs.Remove(s.id)
	}()
	var shutdown <-chan time.Time // Loop termination timer

	for {
		select {
		case req := <-s.add:
			s.clients[req.id] = req.client
		case id := <-s.remove:
			delete(s.clients, id)
			if len(s.clients) < 1 {
				// Close after 10 seconds, if still no clients
				shutdown = time.After(time.Second * 10)
			}
		case buf := <-s.write:
			for _, cl := range s.clients {
				cl <- buf
			}
		case req := <-s.getJSON:
			s.sendJSON(req)
		case <-s.close:
			return
		case <-shutdown:
			if len(s.clients) < 1 {
				return
			}
			shutdown = nil
		}
	}
}

// sendJSON retrieves cached thread post data or reads new data from the
// database, if the cached data is more than 100 messages outdated, then sends
// that data to the requester channel.
func (s *Subscription) sendJSON(req chan<- []byte) {
	if s.data.buffer != nil && s.counter-s.data.counter < 100 {
		req <- s.data.buffer
		return
	}
	thread, err := db.NewReader("", auth.Ident{}).GetThread(s.id, 0)
	if err != nil {
		req <- nil
		return
	}
	s.data.counter = thread.LogCtr
	data, err := json.Marshal(thread)
	if err != nil {
		req <- nil
		return
	}
	req <- data
}

// Close terminates the Subscription
func (s *Subscription) Close(err error) {
	s.close <- err
}
