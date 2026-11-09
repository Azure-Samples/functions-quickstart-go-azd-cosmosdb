package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/azure/azure-functions-golang-worker/sdk"
	"github.com/azure/azure-functions-golang-worker/sdk/bindings"
)

func TestProcessChanges(t *testing.T) {
	document := func(id, data string) bindings.CosmosDocument {
		return bindings.CosmosDocument{ID: id, Data: json.RawMessage(data)}
	}
	for _, tc := range []struct {
		name      string
		documents []bindings.CosmosDocument
		wantError bool
	}{
		{"empty batch", nil, false},
		{"task", []bindings.CosmosDocument{document("task-001", `{"title":"Learn Go","completed":false}`)}, false},
		{"batch", []bindings.CosmosDocument{
			document("task-001", `{"title":"Learn Go","completed":false}`),
			document("task-002", `{"title":"Deploy a function","completed":true}`),
		}, false},
		{"missing id", []bindings.CosmosDocument{document("", `{"title":"Learn Go"}`)}, true},
		{"missing title", []bindings.CosmosDocument{document("task-001", `{}`)}, true},
		{"invalid JSON", []bindings.CosmosDocument{document("task-001", `{`)}, true},
		{"null data", []bindings.CosmosDocument{document("task-001", `null`)}, true},
		{"missing data", []bindings.CosmosDocument{document("task-001", "")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := processChanges(context.Background(), tc.documents)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}

func TestCosmosRegistration(t *testing.T) {
	count := 0
	newFunctionApp().GetRegisteredFunctions().Range(func(_, value any) bool {
		count++
		fn := value.(*sdk.RegisteredFunction)
		binding := fn.TriggerBinding()
		if fn.FuncName != "processChanges" || binding == nil || binding.CosmosDBBinding == nil {
			t.Fatal("missing Cosmos DB trigger")
		}
		b := binding.CosmosDBBinding
		if b.DatabaseName != "ToDoList" || b.ContainerName != "Items" ||
			b.Connection != "CosmosDBConnection" || b.LeaseContainerName != "leases" ||
			b.CreateLeaseContainerIfNotExists || !b.StartFromBeginning {
			t.Fatalf("unexpected change feed configuration: %+v", b)
		}
		if fn.Retry == nil || fn.Retry.MaxRetryCount != 3 ||
			fn.Retry.DelayInterval == nil || *fn.Retry.DelayInterval != 5*time.Second {
			t.Fatalf("unexpected retry configuration: %+v", fn.Retry)
		}
		return true
	})
	if count != 1 {
		t.Fatalf("registered functions = %d, want 1", count)
	}
}
