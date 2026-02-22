package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
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

const releaseInventoryQuery = "UPDATE products SET items_available = items_available \\+ \\$1 WHERE id = \\$2"

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
