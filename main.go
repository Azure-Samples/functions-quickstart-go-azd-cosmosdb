package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/azure/azure-functions-golang-worker/sdk"
	"github.com/azure/azure-functions-golang-worker/sdk/bindings"
	"github.com/azure/azure-functions-golang-worker/worker"
)

type Task struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func main() {
	worker.Start(newFunctionApp())
}

func newFunctionApp() *sdk.App {
	app := sdk.FunctionApp()
	delay := 5 * time.Second
	app.CosmosDB("processChanges", processChanges,
		sdk.WithDatabase("ToDoList"),
		sdk.WithContainer("Items"),
		sdk.WithConnection("CosmosDBConnection"),
		sdk.WithLeaseContainer("leases"),
		sdk.WithCreateLeaseContainerIfNotExists(false),
		sdk.WithStartFromBeginning(true),
		sdk.WithRetry(&sdk.RetryOptions{
			Strategy: sdk.FixedDelay, MaxRetryCount: 3, DelayInterval: &delay,
		}),
	)
	return app
}

func processChanges(ctx context.Context, documents []bindings.CosmosDocument) error {
	for _, document := range documents {
		var task Task
		if err := json.Unmarshal(document.Data, &task); err != nil {
			return fmt.Errorf("decode document %q data: %w", document.ID, err)
		}
		if document.ID == "" || task.Title == "" {
			return fmt.Errorf("document id and data.title are required")
		}
		slog.InfoContext(ctx, "document change processed",
			"document_id", document.ID,
			"title", task.Title,
			"completed", task.Completed,
		)
	}
	return nil
}
