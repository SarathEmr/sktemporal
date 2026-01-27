package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"sktemporal/model"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/activity"
)

const (
	dbConnectionString = "postgres://admin:admin@localhost:5432/appdb?sslmode=disable"
)

// InventoryResult holds the result of inventory update
type InventoryResult struct {
	ProductID        uuid.UUID
	QuantityDeducted int
	OrderID          uuid.UUID
}

// PaymentResult holds the result of payment deduction
type PaymentResult struct {
	OrderID    uuid.UUID
	AmountPaid float64
}

// Activity 1: Update Inventory
func UpdateInventoryActivity(ctx context.Context, request model.OrderRequest) (InventoryResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating inventory", "productID", request.ProductID, "quantity", request.ProductQuantity)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return InventoryResult{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return InventoryResult{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check and update inventory
	var currentStock int
	var price float64
	err = tx.QueryRow(
		"SELECT items_available, price FROM products WHERE id = $1",
		request.ProductID,
	).Scan(&currentStock, &price)
	if err != nil {
		return InventoryResult{}, fmt.Errorf("failed to get product with UUID %s: %w", request.ProductID, err)
	}

	if currentStock < request.ProductQuantity {
		return InventoryResult{}, fmt.Errorf("insufficient stock: available %d, requested %d", currentStock, request.ProductQuantity)
	}

	// Update inventory
	newStock := currentStock - request.ProductQuantity
	_, err = tx.Exec(
		"UPDATE products SET items_available = $1 WHERE id = $2",
		newStock,
		request.ProductID,
	)
	if err != nil {
		return InventoryResult{}, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Create order record
	productsJSON, _ := json.Marshal(map[string]interface{}{
		"productID": request.ProductID.String(),
		"quantity":  request.ProductQuantity,
	})
	totalPrice := price * float64(request.ProductQuantity)

	orderID := uuid.New()
	_, err = tx.Exec(
		`INSERT INTO orders (id, userID, products, total_price, status) 
		 VALUES ($1, $2, $3, $4, $5)`,
		orderID,
		request.UserID,
		productsJSON,
		totalPrice,
		"ADDED_TO_CART",
	)
	if err != nil {
		return InventoryResult{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return InventoryResult{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Inventory updated successfully", "orderID", orderID)
	return InventoryResult{
		ProductID:        request.ProductID,
		QuantityDeducted: request.ProductQuantity,
		OrderID:          orderID,
	}, nil
}

// Compensation Activity: Release Inventory
func ReleaseInventoryActivity(ctx context.Context, result InventoryResult) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing inventory", "productID", result.ProductID, "quantity", result.QuantityDeducted)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Restore inventory
	_, err = db.Exec(
		"UPDATE products SET items_available = items_available + $1 WHERE id = $2",
		result.QuantityDeducted,
		result.ProductID,
	)
	if err != nil {
		logger.Error("Failed to release inventory", "error", err)
		return fmt.Errorf("failed to release inventory: %w", err)
	}

	logger.Info("Inventory released successfully")
	return nil
}

// Activity 2: Deduct Payment
func DeductPaymentActivity(ctx context.Context, request model.OrderRequest, inventoryResult InventoryResult) (PaymentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing payment", "orderID", inventoryResult.OrderID)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Update order status and fetch total price in a single round-trip
	var totalPrice float64
	err = db.QueryRow(
		`UPDATE orders
		 SET status = $1
		 WHERE id = $2
		 RETURNING total_price`,
		"SHIPPING_INITIATED",
		inventoryResult.OrderID,
	).Scan(&totalPrice)
	if err != nil {
		return PaymentResult{}, fmt.Errorf("failed to update order status or fetch total: %w", err)
	}

	// Simulate payment processing
	// In a real scenario, this would call a payment gateway
	time.Sleep(2 * time.Second) // Simulate API call

	logger.Info("Payment processed successfully", "amount", totalPrice)
	return PaymentResult{
		OrderID:    inventoryResult.OrderID,
		AmountPaid: totalPrice,
	}, nil
}

// Compensation Activity: Refund Payment
func RefundPaymentActivity(ctx context.Context, result PaymentResult) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding payment", "orderID", result.OrderID, "amount", result.AmountPaid)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Update order status to indicate payment failure
	_, err = db.Exec(
		`UPDATE orders SET status = $1 WHERE id = $2`,
		"PAYMENT_FAILED",
		result.OrderID,
	)
	if err != nil {
		logger.Error("Failed to update order status for refund", "error", err)
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// In a real scenario, this would call a payment gateway to process the refund
	logger.Info("Payment refunded successfully")
	return nil
}

// Activity 3: Shipping
func ShippingActivity(ctx context.Context, request model.OrderRequest, paymentResult PaymentResult) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing shipping", "orderID", paymentResult.OrderID)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Simulate shipping process
	time.Sleep(2 * time.Second) // Simulate shipping API call

	// Update order status to delivered
	_, err = tx.Exec(
		`UPDATE orders SET status = $1 WHERE id = $2`,
		"ORDER_DELIVERED",
		paymentResult.OrderID,
	)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Shipping completed successfully")
	return nil
}
