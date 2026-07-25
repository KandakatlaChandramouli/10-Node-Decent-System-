package runtime

import (
	"sync"
)

type EventType string

const (
	EventServiceStarted   EventType = "SERVICE_STARTED"
	EventServiceStopped   EventType = "SERVICE_STOPPED"
	EventStateCommitted   EventType = "STATE_COMMITTED"
	EventConsensusProposed EventType = "CONSENSUS_PROPOSED"
	EventFaultInjected    EventType = "FAULT_INJECTED"
)

type Event struct {
	Type    EventType
	Emitter string
	Payload interface{}
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan Event
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
	}
}

func (eb *EventBus) Subscribe(eType EventType) <-chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Event, 100)
	eb.subscribers[eType] = append(eb.subscribers[eType], ch)
	return ch
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subs, ok := eb.subscribers[event.Type]; ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
				// Non-blocking write to prevent slow subscribers from stalling core runtime
			}
		}
	}
}
