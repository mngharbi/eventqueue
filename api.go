package eventqueue

import (
	"sync"
)

/*
	Main API
*/

func New() (eq *EventQueue) {
	// Make eventqueue structure
	eq = &EventQueue{
		listeners:         make(map[string]*listenerRecord),
		closing:           false,
		closingWaitGroup:  &sync.WaitGroup{},
		quitChannel:       make(chan bool, 1),
		nextEventId:       0,
		events:            []*eventRecord{},
		notificationQueue: make(chan bool),
		lock:              &sync.RWMutex{},
	}

	// Closing wait group waits only for remover to be done
	eq.closingWaitGroup.Add(1)

	// Startup remover
	go eq.removeEvents()

	return
}

/*
	Publishes an event
	It will not wait for any listeners to read it before return it
*/
func (eq *EventQueue) Publish(event Event) error {
	eq.lock.Lock()
	defer func() { eq.lock.Unlock() }()

	// Verify it's not closed
	if eq.closing {
		return eventqueueClosed
	}

	// Make event record
	eventRec := &eventRecord{
		id:      eq.nextEventId,
		cond:    sync.NewCond(eq.lock),
		read:    make(map[string]bool),
		payload: event,
	}

	// Add to pubsub tracked events
	eq.events = append(eq.events, eventRec)

	// Update count for next event
	eq.nextEventId += 1

	// Notify all listeners to pick up new event
	for _, listenerRec := range eq.listeners {
		go listenerRec.notifyNewEvents(1)
	}

	// Notify remover
	go eq.notifyRemover()

	return nil
}

/*
	Adds listener
	Returns: listener id (used for unsubscribe), channel for receiving events
*/
func (eq *EventQueue) Subscribe() (eventChannel chan Event, id string) {
	// Generate id
	id = generateSubscriberId()
	eventChannel = make(chan Event)

	// Make listener record
	listenerRec := &listenerRecord{
		id:                id,
		closing:           false,
		quitChannel:       make(chan bool, 1),
		listenerChannel:   eventChannel,
		notificationQueue: make(chan bool),
	}

	eq.lock.Lock()
	defer func() { eq.lock.Unlock() }()

	// Add to listeners map
	eq.listeners[id] = listenerRec

	// Start up drainer
	go eq.drainEvents(listenerRec)

	return
}

func (eq *EventQueue) Unsubscribe(id string) error {
	eq.lock.Lock()
	defer func() { eq.lock.Unlock() }()

	// Get listener record
	listenerRec, ok := eq.listeners[id]
	if !ok {
		return unknownSubscriberId
	}

	// Delete from listeners
	delete(eq.listeners, id)

	// Mark as closing
	listenerRec.closing = true

	// Delete read from all events
	for _, eventRec := range eq.events {
		delete(eventRec.read, id)
	}

	// Push to quit channel to wake up drainer early
	listenerRec.quitChannel <- true

	return nil
}

/*
	Closes event channel
	Prevents new events from being added and closes outgoing channels
*/
func (eq *EventQueue) Done() {
	eq.lock.Lock()
	defer func() { eq.lock.Unlock() }()

	eq.closing = true

	// If there are no events, force remover to wake up
	// Note: There won't be any other events in between because of the flag
	if len(eq.events) == 0 {
		eq.quitChannel <- true
	}
}

/*
	Wait for channel to close and all events to be read
*/
func (eq *EventQueue) Wait() {
	eq.closingWaitGroup.Wait()
}

/*
	Drainer
*/

func (eq *EventQueue) drainEvents(listenerRec *listenerRecord) {
	// Determine initial target event id
	targetId := eq.getInitialTargetId()

	// Wait to be notified of new events
mainloop:
	for {
		select {
		case <-listenerRec.notificationQueue:
			// Read target event
			var eventRec *eventRecord
			eq.lock.RLock()
			firstId := eq.events[0].id
			targetIndex := targetId - firstId
			eventRec = eq.events[targetIndex]
			eq.lock.RUnlock()

			// Feed event to listener
			listenerRec.listenerChannel <- eventRec.payload

			eq.lock.Lock()

			// Skip marking as read if listener was just removed
			if listenerRec.closing {
				eq.lock.Unlock()
				break mainloop
			}

			// Mark event as read
			eventRec.read[listenerRec.id] = true

			// If all listeners are done reading first event, notify remover through condition variable
			if eq.events[0].id == targetId &&
				len(eventRec.read) == len(eq.listeners) {
				eventRec.cond.Signal()
			}

			// If event queue is closing and we're done with events, close channel early
			if eq.closing &&
				eq.events[len(eq.events)-1].id == targetId {
				eq.lock.Unlock()
				break mainloop
			}

			eq.lock.Unlock()

			targetId += 1
		case <-listenerRec.quitChannel:
			break mainloop
		}
	}

	// Once we're done, close subscriber's channel
	close(listenerRec.listenerChannel)
}

/*
	Remover
*/
func (eq *EventQueue) removeEvents() {
mainloop:
	for {
		select {
		case <-eq.notificationQueue:
			eq.lock.Lock()

			eventRec := eq.events[0]

			// Try to win the race against new listeners
			for len(eventRec.read) != len(eq.listeners) {
				eventRec.cond.Wait()
			}

			// Remove first event
			eq.events = eq.events[1:]

			if eq.closing && len(eq.events) == 0 {
				eq.lock.Unlock()
				break mainloop
			}

			eq.lock.Unlock()
		case <-eq.quitChannel:
			break mainloop
		}
	}

	// Remove references
	eq.events = nil
	eq.listeners = nil

	// Unblock closing wait group
	eq.closingWaitGroup.Done()
}
