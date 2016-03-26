package websockets

import (
	"github.com/bakape/dispatcher"
	"github.com/bakape/meguca/util"
	"sync"
)

// Subs is the only instance of SubscriptionMap in this running instance, that
// constains and manages all active subscriptions
var Subs = SubscriptionMap{
	subs: make(map[uint64]*Subscription),
}

// SubscriptionMap contains all active Subscriptions to threads and boards
type SubscriptionMap struct {
	sync.RWMutex
	subs map[uint64]*Subscription
}

// ListenTo assigns a client to listen to the specified subscription. If the
// subscription is not currently active, it is created.
func (s *SubscriptionMap) ListenTo(id uint64, client *Client) {
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
	sub.AddWithID(client.sender, client.ID)
}

func (s *SubscriptionMap) newSubsctiption(id uint64) (*Subscription, error) {
	sub := &Subscription{
		id: id,
		Dispatcher: dispatcher.Dispatcher{
			Listeners: make(dispatcher.ListenerMap),
			IDLength:  16,
		},
	}
	if err := sub.Init(); err != nil {
		return nil, err
	}
	return sub, nil
}

// Unlisten removes a listener from a subscription and removes the subscription,
// if it no longer has any listeners.
func (s *SubscriptionMap) Unlisten(subID uint64, clientID string) {
	s.Lock()
	defer s.Unlock()
	sub := s.subs[subID]
	sub.Remove(clientID)
	if sub.ListenerCount() < 1 {
		delete(s.subs, subID)
	}
}

// Exists returns weather a subscription for a thread already exists
func (s *SubscriptionMap) Exists(id uint64) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.subs[id]
	return ok
}

// Subscription manages and dispatches messages to clients subscribed to the
// thread or board this Subscription instance represents
type Subscription struct {
	id uint64
	dispatcher.Dispatcher
}

// Init reads thread data from the database and prepares the subscription for
// operation
func (s *Subscription) Init() error {
	return nil
}
