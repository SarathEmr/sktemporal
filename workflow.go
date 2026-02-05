package main

import (
	"fmt"
	"time"

	"sktemporal/model"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// CompensationFunc represents a compensation action
type CompensationFunc func(workflow.Context) error

// OrderWorkflow orchestrates the order processing workflow using Saga pattern
func OrderWorkflow(ctx workflow.Context, request model.OrderRequest) (err error) {

	fmt.Println("--- OrderWorkflow started ---")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2, // 1 retry = 2 total attempts
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Track compensations in reverse order (LIFO - Last In First Out)
	var compensations []CompensationFunc

	// Defer compensation execution if an error occurs
	defer func() {
		if err != nil && len(compensations) > 0 {
			fmt.Println("--- Executing compensations in reverse order ---")
			// Execute compensations in reverse order (LIFO)
			for i := len(compensations) - 1; i >= 0; i-- {
				if compErr := compensations[i](ctx); compErr != nil {
					workflow.GetLogger(ctx).Error("Compensation failed", "error", compErr)
					// Continue with other compensations even if one fails
				}
			}
		}
	}()

	fmt.Println("--- Activity Options set ---")

	// Activity 1: Update inventory
	var inventoryResult InventoryResult
	err = workflow.ExecuteActivity(ctx, UpdateInventoryActivity, request).Get(ctx, &inventoryResult)
	if err != nil {
		// Activity failed before being added to saga, no compensation needed
		return err
	}
	// Add compensation step for inventory release
	compensations = append(compensations, func(ctx workflow.Context) error {
		return workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, inventoryResult).Get(ctx, nil)
	})

	fmt.Println("--- Inventory updated ---")

	// Activity 2: Deduct payment
	var paymentResult PaymentResult
	err = workflow.ExecuteActivity(ctx, DeductPaymentActivity, request, inventoryResult).Get(ctx, &paymentResult)
	if err != nil {
		// Error occurred, compensations will be executed by defer
		return err
	}
	// Add compensation step for payment refund
	compensations = append(compensations, func(ctx workflow.Context) error {
		return workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentResult).Get(ctx, nil)
	})

	fmt.Println("--- Payment deducted ---")

	// Activity 3: Shipping
	err = workflow.ExecuteActivity(ctx, ShippingActivity, request, paymentResult).Get(ctx, nil)
	if err != nil {
		// Error occurred, compensations will be executed by defer in reverse order
		return err
	}

	fmt.Println("--- Shipping completed ---")

	// All activities succeeded, no compensation needed
	return nil
}
