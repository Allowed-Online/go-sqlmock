package main

import (
	"fmt"
	"testing"

	"github.com/Allowed-Online/go-sqlmock"
)

// will test that order with a different status cannot be cancelled
func TestShouldNotCancelOrderWithNonPendingStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"o_id", "o_value", "o_reserved_fee", "o_status", "u_id", "u_username", "u_balance"}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM orders AS o INNER JOIN users AS u (.+) FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 25.75, 3.25, 1, 2, "buyer", 10.00))
	mock.ExpectRollback()

	err = cancelOrder(1, db)
	if err != nil {
		t.Errorf("expected no error, but got %s instead", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// will test order cancellation with full refund
func TestShouldRefundUserWhenOrderIsCancelled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"o_id", "o_value", "o_reserved_fee", "o_status", "u_id", "u_username", "u_balance"}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM orders AS o INNER JOIN users AS u (.+) FOR UPDATE").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 25.75, 3.25, 0, 2, "buyer", 10.00))
	// expect user balance update
	mock.ExpectExec("UPDATE users SET balance").
		WithArgs(25.75+3.25, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// expect order status update
	mock.ExpectExec("UPDATE orders SET status").
		WithArgs(ORDER_CANCELLED, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = cancelOrder(1, db)
	if err != nil {
		t.Errorf("expected no error, but got %s instead", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// will test order cancellation rolls back on query error
func TestShouldRollbackOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM orders AS o INNER JOIN users AS u (.+) FOR UPDATE").
		WithArgs(1).
		WillReturnError(fmt.Errorf("some error"))
	mock.ExpectRollback()

	err = cancelOrder(1, db)
	if err == nil {
		t.Error("expected error, but got none")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
