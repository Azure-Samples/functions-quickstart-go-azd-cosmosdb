<!--
---
name: Azure Functions Go Cosmos DB Trigger using Azure Developer CLI
description: Process Cosmos DB change feeds in Go on Azure Functions Flex Consumption using managed identity.
page_type: sample
languages:
- azdeveloper
- go
products:
- azure
- azure-functions
- azure-cosmos-db
urlFragment: functions-quickstart-go-azd-cosmosdb
---
-->

# Azure Functions Go Cosmos DB Change Feed Trigger

This sample follows the [Go timer quickstart](https://github.com/Azure-Samples/functions-quickstart-go-azd-timer)
and deploys to Linux Flex Consumption with `azd`. Go worker packages are preview
packages. The `processChanges` function receives batches of inserts and updates
from `ToDoList/Items` and logs each task:

```json
{
  "id": "task-001",
  "category": "samples",
  "data": {"title": "Learn Go change feeds", "completed": false}
}
```

Task fields are nested under `data` to match the Go SDK's `CosmosDocument.Data`
field. The monitored container's partition key is `/category`; the separate
`leases` container uses `/id`. Infrastructure creates both containers before
the listener starts: identity-based connections cannot create the lease container
automatically.

## Prerequisites

- [Go 1.25 or later](https://go.dev/dl/)
- [Azure Functions Core Tools 4.12 or later](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)
- Azure subscription with permissions to provision resources and assign roles
- [Azurite](https://learn.microsoft.com/azure/storage/common/storage-use-azurite) for local host storage
- Optional: VS Code with the Azure Functions and Go extensions

## Initialize and deploy

```powershell
git clone https://github.com/Azure-Samples/functions-quickstart-go-azd-cosmosdb.git
cd functions-quickstart-go-azd-cosmosdb
az login
azd auth login
azd env new cosmos-dev
azd env set AZURE_PRINCIPAL_ID (az ad signed-in-user show --query id -o tsv)
azd up
```

Infrastructure creates a **serverless Azure Cosmos DB for NoSQL** account,
database and containers, managed identity, host storage, Application Insights,
and a VNet with private endpoints for **host storage** by default. Cosmos DB has
a public endpoint secured by Entra ID; key authentication is disabled.
Serverless Cosmos DB bills for consumed request units and stored data. Choose a
region supporting both Functions Flex Consumption and serverless Cosmos DB.

The app identity has **Cosmos DB Built-in Data Contributor** scoped to
`ToDoList` for reading the change feed and updating leases. The optional test
user receives that data-plane role scoped to `Items`. Azure management-plane
Contributor alone does not grant document access.

To disable the host storage VNet for development:

```shell
azd env set VNET_ENABLED false
azd up
```

## Write a task and verify

The Go writer uses `DefaultAzureCredential`, including your Azure CLI login:

```powershell
$deployment = azd env get-values --output json | ConvertFrom-Json
$env:COSMOS_ENDPOINT = $deployment.COSMOS_ENDPOINT
go run ./cmd/write
```

Each run creates a uniquely identified task. Allow a few minutes for initial
RBAC propagation. In the function app's Application Insights **Logs**, run:

```kusto
traces
| where timestamp > ago(15m)
| where message contains "document change processed"
| project timestamp, message, customDimensions
```

You should see the generated document ID and `title="Learn Go change feeds"`.
Updating that document's `data` fields in Data Explorer also triggers the function.
The latest-version feed observes inserts and updates, not deletes; intermediate
versions may be coalesced.

## Checkpoints, retries, and local development

With no existing leases, this sample starts from the beginning of the feed.
On restart it resumes existing lease checkpoints; `startFromBeginning` does not
reset them. Delivery is at least once, so production side effects must be
idempotent.

Invalid documents return an error and the batch is retried three times with a
five-second fixed delay. Already processed documents in that batch can repeat.
After retries are exhausted processing can advance past the failed batch; there
is no built-in dead-letter queue. Production apps should persist rejected
documents explicitly.

For local development, use the provisioned account and grant your development
identity Data Contributor on the database, not just `Items`, so it can update
leases. The sample writer's container-scoped role is insufficient for a listener.

```powershell
Copy-Item local.settings.example.json local.settings.json
```

Replace `CosmosDBConnection__accountEndpoint` with the account endpoint. Start
Azurite in another terminal, sign in with `az login`, and run `func start`.
The host builds the Go app automatically; no `function.json` is needed.
Azurite emulates host storage only, not Cosmos DB.
Stop the deployed listener or use a separate lease container/prefix when testing
locally: listeners sharing leases divide the feed rather than each receiving
every change.

## Tests, redeployment, and cleanup

```shell
go test ./...
go vet ./...
azd up
azd down
```

Tests cover empty and multi-document batches, task validation, malformed data,
and change-feed registration with pre-provisioned leases and bounded retries.
