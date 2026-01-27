package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func startWorker() {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	if err != nil {
		log.Fatalln("Unable to create temporal client", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, "order-processing-task-queue", worker.Options{})

	// Register workflow
	w.RegisterWorkflow(OrderWorkflow)

	// Register activities
	w.RegisterActivity(UpdateInventoryActivity)
	w.RegisterActivity(ReleaseInventoryActivity)
	w.RegisterActivity(DeductPaymentActivity)
	w.RegisterActivity(RefundPaymentActivity)
	w.RegisterActivity(ShippingActivity)

	// Start worker
	log.Println("Worker started. Press Ctrl+C to exit.")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
