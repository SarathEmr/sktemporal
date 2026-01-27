package main

import (
	"context"
	"log"

	"sktemporal/model"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// Example: How to start the order temporal workflow
func main() {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("Unable to create temporal client", err)
	}
	defer c.Close()

	// Create workflow request using UUIDs from seed data
	request := model.OrderRequest{
		UserID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), // from seed data
		ProductID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"), // from seed data
		ProductQuantity: 2,
	}

	// Start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "order-workflow-" + uuid.New().String(),
		TaskQueue: "order-processing-task-queue",
	}

	// Note: OrderWorkflow is in the main package, so we reference it by name
	// In a real scenario, you might want to move OrderWorkflow to a shared package
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, "OrderWorkfloww", request)
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Printf("Started workflow with ID: %s and RunID: %s\n", we.GetID(), we.GetRunID())

	// Wait for workflow completion (optional)
	var result error
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable to get workflow result", err)
	}

	log.Println("Workflow completed successfully")
}
