package eventqueue

import (
	"github.com/rs/xid"
)

/*
	Listener helpers
*/

// Used to notify a listener of n events
func (listenerRec *listenerRecord) notifyNewEvents (eventsnb int) {
	for i := 0; i < eventsnb; i++ {
		listenerRec.notificationQueue <- true
	}
}

func generateSubscriberId() string {
	return xid.New().String()
}


/*
	Pubsub helpers
*/

func (eq *EventQueue) getInitialTargetId() (targetId int) {
	eq.lock.RLock()
	defer func() { eq.lock.RUnlock() }()
	if len(eq.events) > 0 {
		targetId = eq.events[0].id
	} else {
		targetId = eq.nextEventId
	}
	return
}

func (eq *EventQueue) notifyRemover() {
	eq.notificationQueue <- true
}