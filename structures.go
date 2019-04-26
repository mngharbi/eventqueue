package eventqueue

import (
	"errors"
	"sync"
)

var (
	eventqueueClosed    error = errors.New("Event queue is closed")
	unknownSubscriberId error = errors.New("Unknown subscriber id")
)

// Generic event to be published/received
type Event interface{}

/*
	Pubsub structure
*/

type EventQueue struct {
	listeners map[string]*listenerRecord

	// Used to determine if queue is closing
	closing bool

	// Used to wait for all events to be read
	closingWaitGroup *sync.WaitGroup

	// Used to unblock remover
	quitChannel chan bool

	// Keep track of events
	nextEventId int
	events      []*eventRecord

	// Used to notify remover of first event being done
	notificationQueue chan bool

	lock *sync.RWMutex
}

/*
	Structure to keep track of an event
*/
type eventRecord struct {
	id int

	// Used to wake up remover (used with pubsub lock)
	cond *sync.Cond

	// Contains all listeners that read the event (updated when listener is removed)
	read map[string]bool

	payload Event
}

/*
	Structure to keep track of a listener
*/
type listenerRecord struct {
	id string

	// Used to prevent out of sync event write from waking up remover
	closing bool

	// Used for unsubscribe
	quitChannel chan bool

	// User channel we send events to
	listenerChannel chan Event

	// Used to notify listener that there's a new event
	notificationQueue chan bool
}
