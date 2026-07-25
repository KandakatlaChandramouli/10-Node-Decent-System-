package simulation

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	distconfig "sovereign-chain/applications/dist-config"
	metadataservice "sovereign-chain/applications/metadata-service"
	"sovereign-chain/core/runtime"
	"sovereign-chain/modules/storage"
)

func TestServiceRegistryAndScheduler(t *testing.T) {
	reg := runtime.NewServiceRegistry()
	memStore := storage.NewMemoryStorage()

	if err := reg.Register(memStore); err != nil {
		t.Fatalf("Service registration failed: %v", err)
	}

	healths := reg.HealthCheckAll()
	if err, ok := healths["Storage-InMemory"]; !ok || err != nil {
		t.Fatalf("Service health check failed: %v", err)
	}

	sched := runtime.NewScheduler()
	var executed atomic.Bool
	var timeMu sync.Mutex
	var lastExec time.Time

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	sched.ScheduleRecurring(ctx, "health_monitor", 20*time.Millisecond, func(c context.Context) error {
		executed.Store(true)
		timeMu.Lock()
		lastExec = time.Now()
		timeMu.Unlock()
		return nil
	})

	time.Sleep(50 * time.Millisecond)
	sched.Stop()

	timeMu.Lock()
	execTime := lastExec
	timeMu.Unlock()

	if !executed.Load() || execTime.IsZero() {
		t.Fatalf("Scheduler failed to execute recurring task")
	}

	t.Logf("Runtime Service Registry & Scheduler verified successfully without race conditions")
}

func TestDistributedConfigService(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	configApp := distconfig.NewConfigState(memStore)

	setPayload, _ := json.Marshal(distconfig.ConfigPayload{
		Action:    distconfig.ActionSetConfig,
		Namespace: "prod_db",
		Key:       "max_connections",
		Value:     "500",
	})

	root, err := configApp.Apply(setPayload)
	if err != nil {
		t.Fatalf("Failed to set configuration: %v", err)
	}

	val, found := configApp.GetConfig("prod_db", "max_connections")
	if !found || val != "500" {
		t.Fatalf("Expected max_connections=500, got %s", val)
	}

	t.Logf("Distributed Config Service verified successfully. Root: %s", string(root))
}

func TestMetadataServiceApplication(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	metaApp := metadataservice.NewMetadataState(memStore)

	metaPayload, _ := json.Marshal(metadataservice.MetadataPayload{
		Action:   metadataservice.ActionPutMeta,
		ObjectID: "obj_block_001",
		Metadata: map[string]string{
			"owner": "system",
			"size":  "2048",
		},
	})

	root, err := metaApp.Apply(metaPayload)
	if err != nil {
		t.Fatalf("Failed to apply metadata: %v", err)
	}

	meta, found := metaApp.GetMetadata("obj_block_001")
	if !found || meta["owner"] != "system" {
		t.Fatalf("Metadata retrieval failed. Expected owner=system")
	}

	t.Logf("Metadata Service verified successfully. Root: %s", string(root))
}
