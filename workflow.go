package main

import (
	"fmt"
	"time"

	"sktemporal/model"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OrderWorkflow orchestrates the order processing workflow using Saga pattern
func OrderWorkflow(ctx workflow.Context, request model.OrderRequest) error {

	fmt.Println("--- OrderWorkflow started ---")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2, // 1 retry = 2 total attempts
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	opts := workflow.SagaOptions{}
	saga := workflow.NewSaga(ctx, opts)

	fmt.Println("--- Activity Options set ---")

	// Activity 1: Update inventory
	var inventoryResult InventoryResult
	err := workflow.ExecuteActivity(ctx, UpdateInventoryActivity, request).Get(ctx, &inventoryResult)
	if err != nil {
		// Activity failed before being added to saga, no compensation needed
		return err
	}
	// Add compensation step for inventory release
	saga.AddCompensation(ctx, ReleaseInventoryActivity, inventoryResult)

	fmt.Println("--- Inventory updated ---")

	// Activity 2: Deduct payment
	var paymentResult PaymentResult
	err = workflow.ExecuteActivity(ctx, DeductPaymentActivity, request, inventoryResult).Get(ctx, &paymentResult)
	if err != nil {
		return err
	}
	saga.AddCompensation(ctx, RefundPaymentActivity, paymentResult)

	fmt.Println("--- Payment deducted ---")

	// Activity 3: Shipping
	err = workflow.ExecuteActivity(ctx, ShippingActivity, request, paymentResult).Get(ctx, nil)
	if err != nil {
		// Error occurred, saga will automatically execute compensations in reverse order
		return err
	}

	fmt.Println("--- Shipping completed ---")

	return nil
}
