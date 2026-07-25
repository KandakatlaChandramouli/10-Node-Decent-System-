package simulation

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	distlock "sovereign-chain/applications/dist-lock"
	distqueue "sovereign-chain/applications/dist-queue"
	"sovereign-chain/core/runtime"
	"sovereign-chain/modules/storage"
)

func TestEventBusPubSub(t *testing.T) {
	eb := runtime.NewEventBus()
	sub := eb.Subscribe(runtime.EventStateCommitted)

	go func() {
		eb.Publish(runtime.Event{
			Type:    runtime.EventStateCommitted,
			Emitter: "test_node",
			Payload: "state_root_0x123",
		})
	}()

	select {
	case event := <-sub:
		if event.Emitter != "test_node" || event.Payload != "state_root_0x123" {
			t.Fatalf("Event payload mismatch: %+v", event)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Event bus timed out waiting for event")
	}
}

func TestDistributedLockService(t *testing.T) {
	dir := "./test_data_lock"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	lockApp := distlock.NewLockState(store)

	acqPayload, _ := json.Marshal(distlock.LockPayload{
		Action:   distlock.ActionAcquire,
		Resource: "db_writer_lease",
		Holder:   "node_alpha",
		TTLMs:    5000,
	})

	_, err = lockApp.Apply(acqPayload)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !lockApp.IsLocked("db_writer_lease") {
		t.Fatalf("Expected resource db_writer_lease to be locked")
	}

	relPayload, _ := json.Marshal(distlock.LockPayload{
		Action:   distlock.ActionRelease,
		Resource: "db_writer_lease",
		Holder:   "node_alpha",
	})

	root2, err := lockApp.Apply(relPayload)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	if lockApp.IsLocked("db_writer_lease") {
		t.Fatalf("Expected resource db_writer_lease to be unlocked")
	}

	t.Logf("Distributed Lock Service executed successfully. Root: %s", string(root2))
}

func TestDistributedQueueService(t *testing.T) {
	dir := "./test_data_queue"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	queueApp := distqueue.NewQueueState(store)

	enqPayload, _ := json.Marshal(distqueue.QueuePayload{
		Action: distqueue.ActionEnqueue,
		Topic:  "jobs",
		Data:   []byte("process_payment_#1001"),
	})

	_, err = queueApp.Apply(enqPayload)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	if queueApp.Depth("jobs") != 1 {
		t.Fatalf("Expected queue depth 1, got %d", queueApp.Depth("jobs"))
	}

	deqPayload, _ := json.Marshal(distqueue.QueuePayload{
		Action: distqueue.ActionDequeue,
		Topic:  "jobs",
	})

	root, err := queueApp.Apply(deqPayload)
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}

	if queueApp.Depth("jobs") != 0 {
		t.Fatalf("Expected queue depth 0, got %d", queueApp.Depth("jobs"))
	}

	t.Logf("Distributed Queue Service executed successfully. Root: %s", string(root))
}
