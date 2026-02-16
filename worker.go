package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func startWorker() {
	// Load config from environment (e.g. .env.dev or system env)
	cfg := LoadConfigFromEnv()

	// Get Temporal address from environment variable, default to localhost:7233
	temporalAddress := "localhost:7233"
	if addr := os.Getenv("TEMPORAL_ADDRESS"); addr != "" {
		temporalAddress = addr
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalAddress,
	})
	if err != nil {
		log.Fatalln("Unable to create temporal client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "order-processing-task-queue", worker.Options{})

	// Register workflow
	w.RegisterWorkflow(OrderWorkflow)

	// Register activities with config (pass the Activities instance)
	activities := NewActivities(cfg)

	w.RegisterActivity(activities.UpdateInventoryActivity)
	w.RegisterActivity(activities.ReleaseInventoryActivity)
	w.RegisterActivity(activities.DeductPaymentActivity)
	w.RegisterActivity(activities.RefundPaymentActivity)
	w.RegisterActivity(activities.ShippingActivity)

	// Start worker
	log.Println("Worker started. Press Ctrl+C to exit.")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
