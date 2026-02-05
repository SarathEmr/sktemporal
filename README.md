# Temporal Order Processing Workflow

This project implements a Temporal workflow for processing orders with three activities and compensation logic.

## Workflow Structure

### Request Format
```json
{
  "userID": "uuid",
  "productid": "uuid",
  "productQuantity": int
}
```

### Activities

1. **Update Inventory** (Activity 1)
   - Deducts product quantity from inventory
   - Updates the `product` table
   - Creates an order record in the `order` table
   - **Compensation**: Releases inventory back if workflow fails
   - **Retry**: 1 retry on failure (2 total attempts)

2. **Deduct Payment** (Activity 2)
   - Processes payment for the order
   - Updates the `order` table status
   - **Compensation**: Refunds payment if workflow fails
   - **Retry**: 1 retry on failure (2 total attempts)

3. **Shipping** (Activity 3)
   - Processes shipping for the order
   - Updates the `order` table status to delivered
   - **Retry**: 1 retry on failure (2 total attempts)

## Setup

### Prerequisites
- Docker and Docker Compose
- Go 1.21 or later

### Database Migration

Before running the workflow, apply the UUID migration:

```bash
# The migration file is in postgres-init/03-add-uuid-columns.sql
# It will be automatically applied when you start the database
```

### Start Services

```bash
docker-compose up -d
```

This starts:
- PostgreSQL on port 5432
- Temporal server on port 7233
- Temporal Web UI on port 8088

### Install Dependencies

```bash
go mod download
```

### Run Worker

```bash
go run .
```

The worker will start and listen for workflow executions on the `order-processing-task-queue`.

### Start a Workflow

Use the example in `client_example.go` or create your own client:

```go
c, _ := client.Dial(client.Options{HostPort: "localhost:7233"})
defer c.Close()

request := OrderRequest{
    UserID:          uuid.MustParse("your-user-uuid"),
    ProductID:       uuid.MustParse("your-product-uuid"),
    ProductQuantity: 5,
}

workflowOptions := client.StartWorkflowOptions{
    ID:        "order-workflow-" + uuid.New().String(),
    TaskQueue: "order-processing-task-queue",
}

we, _ := c.ExecuteWorkflow(context.Background(), workflowOptions, OrderWorkflow, request)
```

## Compensation Logic

The workflow implements automatic compensation:

- If Activity 1 fails: No compensation needed (nothing committed)
- If Activity 2 fails: Refunds payment and releases inventory
- If Activity 3 fails: Refunds payment and releases inventory

## Database Schema Notes

- The `product` table requires a `uuid` column (added via migration)
- The `order` table's `userID` column is updated to support UUID strings
- Ensure products have UUIDs populated before running workflows

## Monitoring

Access Temporal Web UI at: http://localhost:8088

You can view workflow executions, activity results, and retry attempts in the UI.


***

When the postgres container gets built the first time, the migrations and seeds gets applied.
The same files won't get applied the second time onwards.

## Steps

$ docker-compose build  
$ docker-compose up  

Connect to Postgres CLI:
$ psql -h localhost -p 5432 -U admin -d temporal      // password = "admin"  
\dt	// list the tables  
\c appdb	// move to a different database  
$ select * from users;  
$ select * from products;  
$ select * from orders;  

Check localhost:8088 for temporal-UI  

Run the task:
// order placing, inventory update, payment deduction
// temporal client submits task to the worker
sktemporal\client> go run main.go  

Check `temporal-worker` logs  

Check db:
// make sure order is placed successfully
$ select * from orders;  
