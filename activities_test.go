package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	"sktemporal/model"
)

type ActivitiesTestSuite struct {
	suite.Suite
	*testsuite.WorkflowTestSuite
}

func TestActivitiesTestSuite(t *testing.T) {
	suite.Run(t, &ActivitiesTestSuite{
		WorkflowTestSuite: &testsuite.WorkflowTestSuite{},
	})
}

const (
	releaseInventoryQuery    = "UPDATE products SET items_available = items_available \\+ \\$1 WHERE id = \\$2"
	selectProductQuery       = "SELECT items_available, price FROM products WHERE id = \\$1"
	updateProductQuery       = "UPDATE products SET items_available = \\$1 WHERE id = \\$2"
	insertOrderQuery         = "INSERT INTO orders \\(id, userID, products, total_price, status\\)"
	deductPaymentUpdateQuery = "UPDATE orders.*SET status.*RETURNING total_price"
	updateOrderStatusQuery   = "UPDATE orders SET status = \\$1 WHERE id = \\$2"
)

func (s *ActivitiesTestSuite) TestUpdateInventoryActivity_OpenDBFailure_ReturnsConnectError() {
	connectErr := errors.New("driver: bad connection")
	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return nil, connectErr }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{
		UserID:          uuid.New(),
		ProductID:       uuid.New(),
		ProductQuantity: 2,
	}

	_, err := env.ExecuteActivity(activities.UpdateInventoryActivity, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to connect to database")
	s.Require().Contains(err.Error(), "driver: bad connection")
}

func (s *ActivitiesTestSuite) TestUpdateInventoryActivity_BeginTxFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	mock.ExpectBegin().WillReturnError(errors.New("tx begin failed"))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{
		UserID:          uuid.New(),
		ProductID:       uuid.New(),
		ProductQuantity: 1,
	}

	_, err = env.ExecuteActivity(activities.UpdateInventoryActivity, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to begin transaction")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestUpdateInventoryActivity_SelectProductFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	productID := uuid.New()
	userID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(selectProductQuery).
		WithArgs(productID).
		WillReturnError(errors.New("product not found"))
	mock.ExpectRollback()

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{
		UserID:          userID,
		ProductID:       productID,
		ProductQuantity: 1,
	}

	_, err = env.ExecuteActivity(activities.UpdateInventoryActivity, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to get product")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestUpdateInventoryActivity_InsufficientStock_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	productID := uuid.New()
	userID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery(selectProductQuery).
		WithArgs(productID).
		WillReturnRows(sqlmock.NewRows([]string{"items_available", "price"}).AddRow(1, 100.0))
	mock.ExpectRollback()

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{
		UserID:          userID,
		ProductID:       productID,
		ProductQuantity: 5,
	}

	_, err = env.ExecuteActivity(activities.UpdateInventoryActivity, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "insufficient stock")
	s.Require().Contains(err.Error(), "available 1, requested 5")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestUpdateInventoryActivity_Success_ReturnsInventoryResult() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	productID := uuid.New()
	userID := uuid.New()
	quantity := 2
	currentStock := 10
	price := 100.0
	newStock := currentStock - quantity

	mock.ExpectBegin()
	mock.ExpectQuery(selectProductQuery).
		WithArgs(productID).
		WillReturnRows(sqlmock.NewRows([]string{"items_available", "price"}).AddRow(currentStock, price))
	mock.ExpectExec(updateProductQuery).
		WithArgs(newStock, productID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	productsJSON, _ := json.Marshal(map[string]interface{}{
		"productID": productID.String(),
		"quantity":  quantity,
	})
	mock.ExpectExec(insertOrderQuery).
		WithArgs(sqlmock.AnyArg(), userID, productsJSON, price*float64(quantity), "ADDED_TO_CART").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{
		UserID:          userID,
		ProductID:       productID,
		ProductQuantity: quantity,
	}

	encoded, err := env.ExecuteActivity(activities.UpdateInventoryActivity, request)
	s.Require().NoError(err)
	s.Require().NoError(mock.ExpectationsWereMet())

	var result InventoryResult
	s.Require().NoError(encoded.Get(&result))
	s.Require().Equal(productID, result.ProductID)
	s.Require().Equal(quantity, result.QuantityDeducted)
	s.Require().NotEqual(uuid.Nil, result.OrderID)
}

func (s *ActivitiesTestSuite) TestReleaseInventoryActivity_OpenDBFailure_ReturnsConnectError() {
	connectErr := errors.New("driver: bad connection")
	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return nil, connectErr }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	result := InventoryResult{
		ProductID:        uuid.New(),
		QuantityDeducted: 1,
		OrderID:          uuid.New(),
	}

	_, err := env.ExecuteActivity(activities.ReleaseInventoryActivity, result)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to connect to database")
	s.Require().Contains(err.Error(), "driver: bad connection")
}

func (s *ActivitiesTestSuite) TestReleaseInventoryActivity_DBQueryFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	productID := uuid.New()
	orderID := uuid.New()
	quantity := 3
	mock.ExpectExec(releaseInventoryQuery).
		WithArgs(quantity, productID).
		WillReturnError(errors.New("connection reset by peer"))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	result := InventoryResult{
		ProductID:        productID,
		QuantityDeducted: quantity,
		OrderID:          orderID,
	}

	_, err = env.ExecuteActivity(activities.ReleaseInventoryActivity, result)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to release inventory")
	s.Require().Contains(err.Error(), "connection reset by peer")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestReleaseInventoryActivity_Success_ReturnsNil() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	productID := uuid.New()
	orderID := uuid.New()
	quantity := 2
	mock.ExpectExec(releaseInventoryQuery).
		WithArgs(quantity, productID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	result := InventoryResult{
		ProductID:        productID,
		QuantityDeducted: quantity,
		OrderID:          orderID,
	}

	_, err = env.ExecuteActivity(activities.ReleaseInventoryActivity, result)
	s.Require().NoError(err)
	s.Require().NoError(mock.ExpectationsWereMet())
}

// --- DeductPaymentActivity ---

func (s *ActivitiesTestSuite) TestDeductPaymentActivity_OpenDBFailure_ReturnsConnectError() {
	connectErr := errors.New("driver: bad connection")
	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return nil, connectErr }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	invResult := InventoryResult{ProductID: uuid.New(), QuantityDeducted: 1, OrderID: uuid.New()}

	_, err := env.ExecuteActivity(activities.DeductPaymentActivity, request, invResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to connect to database")
}

func (s *ActivitiesTestSuite) TestDeductPaymentActivity_UpdateOrderFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	mock.ExpectQuery(deductPaymentUpdateQuery).
		WithArgs("SHIPPING_INITIATED", orderID).
		WillReturnError(errors.New("order not found"))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	invResult := InventoryResult{ProductID: uuid.New(), QuantityDeducted: 1, OrderID: orderID}

	_, err = env.ExecuteActivity(activities.DeductPaymentActivity, request, invResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to update order status or fetch total")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestDeductPaymentActivity_Success_ReturnsPaymentResult() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	totalPrice := 199.99
	mock.ExpectQuery(deductPaymentUpdateQuery).
		WithArgs("SHIPPING_INITIATED", orderID).
		WillReturnRows(sqlmock.NewRows([]string{"total_price"}).AddRow(totalPrice))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	invResult := InventoryResult{ProductID: uuid.New(), QuantityDeducted: 1, OrderID: orderID}

	encoded, err := env.ExecuteActivity(activities.DeductPaymentActivity, request, invResult)
	s.Require().NoError(err)
	s.Require().NoError(mock.ExpectationsWereMet())

	var result PaymentResult
	s.Require().NoError(encoded.Get(&result))
	s.Require().Equal(orderID, result.OrderID)
	s.Require().Equal(totalPrice, result.AmountPaid)
}

// --- RefundPaymentActivity ---

func (s *ActivitiesTestSuite) TestRefundPaymentActivity_OpenDBFailure_ReturnsConnectError() {
	connectErr := errors.New("driver: bad connection")
	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return nil, connectErr }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	paymentResult := PaymentResult{OrderID: uuid.New(), AmountPaid: 99.99}

	_, err := env.ExecuteActivity(activities.RefundPaymentActivity, paymentResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to connect to database")
}

func (s *ActivitiesTestSuite) TestRefundPaymentActivity_UpdateFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	mock.ExpectExec(updateOrderStatusQuery).
		WithArgs("PAYMENT_FAILED", orderID).
		WillReturnError(errors.New("update failed"))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	paymentResult := PaymentResult{OrderID: orderID, AmountPaid: 99.99}

	_, err = env.ExecuteActivity(activities.RefundPaymentActivity, paymentResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to update order status")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestRefundPaymentActivity_Success_ReturnsNil() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	mock.ExpectExec(updateOrderStatusQuery).
		WithArgs("PAYMENT_FAILED", orderID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	paymentResult := PaymentResult{OrderID: orderID, AmountPaid: 99.99}

	_, err = env.ExecuteActivity(activities.RefundPaymentActivity, paymentResult)
	s.Require().NoError(err)
	s.Require().NoError(mock.ExpectationsWereMet())
}

// --- ShippingActivity ---

func (s *ActivitiesTestSuite) TestShippingActivity_OpenDBFailure_ReturnsConnectError() {
	connectErr := errors.New("driver: bad connection")
	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return nil, connectErr }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	paymentResult := PaymentResult{OrderID: uuid.New(), AmountPaid: 199.99}

	_, err := env.ExecuteActivity(activities.ShippingActivity, request, paymentResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to connect to database")
}

func (s *ActivitiesTestSuite) TestShippingActivity_UpdateFailure_ReturnsError() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	mock.ExpectExec(updateOrderStatusQuery).
		WithArgs("ORDER_DELIVERED", orderID).
		WillReturnError(errors.New("update failed"))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	paymentResult := PaymentResult{OrderID: orderID, AmountPaid: 199.99}

	_, err = env.ExecuteActivity(activities.ShippingActivity, request, paymentResult)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to update order status")
	s.Require().NoError(mock.ExpectationsWereMet())
}

func (s *ActivitiesTestSuite) TestShippingActivity_Success_ReturnsNil() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)
	defer db.Close()

	orderID := uuid.New()
	mock.ExpectExec(updateOrderStatusQuery).
		WithArgs("ORDER_DELIVERED", orderID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	oldOpen := openDB
	openDB = func(_, _ string) (*sql.DB, error) { return db, nil }
	defer func() { openDB = oldOpen }()

	activities := NewActivities(&Config{})
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(activities)

	request := model.OrderRequest{UserID: uuid.New(), ProductID: uuid.New(), ProductQuantity: 1}
	paymentResult := PaymentResult{OrderID: orderID, AmountPaid: 199.99}

	_, err = env.ExecuteActivity(activities.ShippingActivity, request, paymentResult)
	s.Require().NoError(err)
	s.Require().NoError(mock.ExpectationsWereMet())
}
