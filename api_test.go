package eventqueue

import (
	"sync"
	"testing"
)

/*
	Helpers
*/

func publish(eq *EventQueue, wg *sync.WaitGroup, nb int) {
	for i := 1; i <= nb; i++ {
		eq.Publish(i)
	}
	wg.Done()
}

func startPublishersAndWait(eq *EventQueue, nbPublishers int, n int) {
	wg := &sync.WaitGroup{}
	wg.Add(nbPublishers)
	for i := 0; i < nbPublishers; i++ {
		go publish(eq, wg, n)
	}
	wg.Wait()
}

func TestOneSubscriber(t *testing.T) {
	eq := New()

	// One Subscriber
	channel, _ := eq.Subscribe()

	// Concurrent publishers
	nbPublishers := 5
	n := 500
	startPublishersAndWait(eq, nbPublishers, n)

	// Close event queue
	go eq.Done()

	// Read until channel closes
	sum := 0
	for nb := range channel {
		sum += nb.(int)
	}

	// Wait for all events to be read (should be done because we waited for channel)
	eq.Wait()

	// Expect sum
	expectedSum := nbPublishers * (n * (n + 1)) / 2
	if sum != expectedSum {
		t.Errorf("Sum with one subscriber does not match expected sum. sum=%v, expected=%v", sum, expectedSum)
	}

	// Make sure struct was cleaned up
	if len(eq.events) != 0 ||
		len(eq.listeners) != 0 {
		t.Errorf("Need to clean up after event queue is done")
	}
}

func TestOnePublisher(t *testing.T) {
	eq := New()

	// 2 Subscribers
	channel1, _ := eq.Subscribe()
	channel2, _ := eq.Subscribe()
	channels := [](chan Event){channel1, channel2}

	// Concurrent publishers
	nbPublishers := 1
	n := 100
	startPublishersAndWait(eq, nbPublishers, n)

	// Close event queue
	go eq.Done()

	for _, channel := range channels {
		// Read until channel closes
		last := 0
		for nb := range channel {
			// Expect numbers to be ordered
			if nb.(int) <= last {
				t.Error("Values should be received in the order they were pushed")
			}
			last = nb.(int)
		}
	}

	// Wait for all events to be read (should be done because we waited for channel)
	eq.Wait()

	// Make sure struct was cleaned up
	if len(eq.events) != 0 ||
		len(eq.listeners) != 0 {
		t.Errorf("Need to clean up after event queue is done")
	}
}

func TestUnsubscribeMultiple(t *testing.T) {
	eq := New()

	// 2 Subscribers
	channel1, _ := eq.Subscribe()
	channel2, id2 := eq.Subscribe()

	// Concurrent publishers
	nbPublishers := 2
	n := 100
	startPublishersAndWait(eq, nbPublishers, n)

	// Unsub second channel
	err := eq.Unsubscribe(id2)
	if err != nil {
		t.Errorf("Unsubscribe should not fail")
	}

	// Make sure listener was removed
	eq.lock.Lock()
	if len(eq.listeners) != 1 {
		t.Errorf("Need to remove listener")
	}
	eq.lock.Unlock()

	// Make sure second channel closes before we close event queue
	for range channel2 {
	}

	// Close event queue
	go eq.Done()

	// Read until channel closes
	sum := 0
	for nb := range channel1 {
		sum += nb.(int)
	}

	// Wait for all events to be read (should be done because we waited for channel)
	eq.Wait()

	// Make sure first channel still reads everything
	expectedSum := nbPublishers * (n * (n + 1)) / 2
	if sum != expectedSum {
		t.Errorf("Sum with one subscriber does not match expected sum. sum=%v, expected=%v", sum, expectedSum)
	}

	// Make sure struct was cleaned up
	if len(eq.events) != 0 ||
		len(eq.listeners) != 0 {
		t.Errorf("Need to clean up after event queue is done")
	}
}

func TestOpenClose(t *testing.T) {
	eq := New()
	eq.Done()
	eq.Wait()
}

func TestPublishClosed(t *testing.T) {
	eq := New()

	// Close queue
	eq.Done()

	if err := eq.Publish(1); err != eventqueueClosed {
		t.Error("Publishing to closed event queue should fail")
	}

	eq.Wait()
}

func TestUnknownUnsubscribe(t *testing.T) {
	eq := New()

	if err := eq.Unsubscribe("ID"); err != unknownSubscriberId {
		t.Error("Unsubscribing with unknown id should fail")
	}

	eq.Done()
	eq.Wait()
}

func TestWaitUntilEmpty(t *testing.T) {
	eq := New()

	// Subscribe
	channel, _ := eq.Subscribe()

	// Make sure wait until empty returns before publishing
	eq.WaitUntilEmpty()
	eq.Publish(0)

	// Pass through empty events waiter
	emptyNotification := make(chan bool)
	go func() {
		eq.WaitUntilEmpty()
		emptyNotification <- true
	}()

	// Read value then wait until empty
	select {
	case <-emptyNotification:
		t.Error("WaitUntilEmpty should not return before events are drained.")
	default:
		break
	}
	<-channel
	<-emptyNotification
}
