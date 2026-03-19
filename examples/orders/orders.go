package main

import (
	"database/sql"
	"log"
)

const ORDER_PENDING = 0
const ORDER_CANCELLED = 1

type User struct {
	Id       int
	Username string
	Balance  float64
}

type Order struct {
	Id          int
	Value       float64
	ReservedFee float64
	Status      int
}

func cancelOrder(id int, db *sql.DB) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	var order Order
	var user User

	// fetch order and buyer in a single row
	err = tx.QueryRow(`
		SELECT o.id, o.value, o.reserved_fee, o.status,
		       u.id, u.username, u.balance
		FROM orders AS o
		INNER JOIN users AS u ON o.buyer_id = u.id
		WHERE o.id = ?
		FOR UPDATE`, id).Scan(
		&order.Id, &order.Value, &order.ReservedFee, &order.Status,
		&user.Id, &user.Username, &user.Balance,
	)
	if err != nil {
		tx.Rollback()
		return
	}

	// ensure order status
	if order.Status != ORDER_PENDING {
		tx.Rollback()
		return
	}

	// refund order value
	_, err = tx.Exec(
		"UPDATE users SET balance = balance + ? WHERE id = ?",
		order.Value+order.ReservedFee, user.Id,
	)
	if err != nil {
		tx.Rollback()
		return
	}

	// update order status
	_, err = tx.Exec(
		"UPDATE orders SET status = ?, updated = NOW() WHERE id = ?",
		ORDER_CANCELLED, order.Id,
	)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

func main() {
	// @NOTE: the real connection is not required for tests
	db, err := sql.Open("mysql", "root:@/orders")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = cancelOrder(1, db)
	if err != nil {
		log.Fatal(err)
	}
}
