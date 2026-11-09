package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	endpoint := os.Getenv("COSMOS_ENDPOINT")
	if endpoint == "" {
		return fmt.Errorf("set COSMOS_ENDPOINT to the account's HTTPS endpoint")
	}
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return err
	}
	client, err := azcosmos.NewClient(endpoint, credential, nil)
	if err != nil {
		return err
	}
	container, err := client.NewContainer("ToDoList", "Items")
	if err != nil {
		return err
	}
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	document, err := json.Marshal(map[string]any{
		"id":       id,
		"category": "samples",
		"data": map[string]any{
			"title":     "Learn Go change feeds",
			"completed": false,
		},
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = container.CreateItem(ctx, azcosmos.NewPartitionKeyString("samples"), document, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Created %s in ToDoList/Items.\n", id)
	return nil
}
