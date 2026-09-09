package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type InventoryOperation string

const (
	InventoryReserve   InventoryOperation = "reserve"
	InventoryRelease   InventoryOperation = "release"
	InventoryConfirm   InventoryOperation = "confirm"
	InventoryUnconfirm InventoryOperation = "unconfirm"
)

type PointsOperation string

const (
	PointsDebit  PointsOperation = "debit"
	PointsRefund PointsOperation = "refund"
)

// RunLocalMutation gives Kitex mutation methods the same atomic SQL path as
// DTM callbacks. Only MySQL deadlocks are retried; business failures are final.
func RunLocalMutation(ctx context.Context, db *sql.DB, operation func(*sql.Tx) error) error {
	if db == nil {
		return errors.New("nil mutation database")
	}
	var lastErr error
	for attempt := 0; attempt < branchMaxAttempts(); attempt++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		err = operation(tx)
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		if !isMySQLDeadlock(err) {
			return err
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * time.Millisecond)
	}
	return fmt.Errorf("mutation retry attempts exhausted: %w", lastErr)
}

func ApplyInventoryMutation(tx *sql.Tx, p SagaPayload, operation InventoryOperation) error {
	switch operation {
	case InventoryReserve:
		return reserveInventoryTx(tx, p)
	case InventoryRelease:
		return releaseInventoryTx(tx, p)
	case InventoryConfirm:
		return confirmInventoryTx(tx, p)
	case InventoryUnconfirm:
		return unconfirmInventoryTx(tx, p)
	default:
		return fmt.Errorf("unknown inventory operation %q", operation)
	}
}

func reserveInventoryTx(tx *sql.Tx, p SagaPayload) error {
	if p.OrderID == "" || p.ProductID <= 0 || p.Quantity <= 0 {
		return errors.New("invalid inventory mutation")
	}
	insert, err := tx.Exec(`INSERT IGNORE INTO mall_inventory_reservations(order_id,product_id,quantity,status,created_at,updated_at) VALUES(?,?,?,?,NOW(),NOW())`, p.OrderID, p.ProductID, p.Quantity, ReservationReserved)
	if err != nil {
		return err
	}
	if affected, _ := insert.RowsAffected(); affected == 0 {
		var productID, quantity int64
		var status string
		if err = tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, p.OrderID).Scan(&productID, &quantity, &status); err != nil {
			return err
		}
		if productID != p.ProductID || quantity != p.Quantity {
			return ErrIdempotencyConflict
		}
		if status == ReservationReserved || status == ReservationConfirmed {
			return nil
		}
		return fmt.Errorf("reservation already %s", status)
	}
	result, err := tx.Exec(`UPDATE mall_inventories SET available=available-?,reserved=reserved+?,version=version+1,updated_at=NOW() WHERE product_id=? AND available>=?`, p.Quantity, p.Quantity, p.ProductID, p.Quantity)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrInsufficientStock
	}
	return nil
}

func releaseInventoryTx(tx *sql.Tx, p SagaPayload) error {
	productID, quantity, status, err := lockReservation(tx, p.OrderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if productID != p.ProductID || quantity != p.Quantity {
		return ErrIdempotencyConflict
	}
	if status == ReservationReleased {
		return nil
	}
	if status != ReservationReserved {
		return fmt.Errorf("cannot release reservation in status %s", status)
	}
	result, err := tx.Exec(`UPDATE mall_inventories SET available=available+?,reserved=reserved-?,version=version+1,updated_at=NOW() WHERE product_id=? AND reserved>=?`, quantity, quantity, productID, quantity)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("inventory invariant violation while releasing")
	}
	_, err = tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=? AND status=?`, ReservationReleased, p.OrderID, ReservationReserved)
	return err
}

func confirmInventoryTx(tx *sql.Tx, p SagaPayload) error {
	productID, quantity, status, err := lockReservation(tx, p.OrderID)
	if err != nil {
		return err
	}
	if productID != p.ProductID || quantity != p.Quantity {
		return ErrIdempotencyConflict
	}
	if status == ReservationConfirmed {
		return nil
	}
	if status != ReservationReserved {
		return fmt.Errorf("cannot confirm reservation in status %s", status)
	}
	result, err := tx.Exec(`UPDATE mall_inventories SET reserved=reserved-?,sold=sold+?,version=version+1,updated_at=NOW() WHERE product_id=? AND reserved>=?`, quantity, quantity, productID, quantity)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("inventory invariant violation while confirming")
	}
	_, err = tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=? AND status=?`, ReservationConfirmed, p.OrderID, ReservationReserved)
	return err
}

func unconfirmInventoryTx(tx *sql.Tx, p SagaPayload) error {
	productID, quantity, status, err := lockReservation(tx, p.OrderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if productID != p.ProductID || quantity != p.Quantity {
		return ErrIdempotencyConflict
	}
	if status == ReservationReserved || status == ReservationReleased {
		return nil
	}
	if status != ReservationConfirmed {
		return fmt.Errorf("cannot unconfirm reservation in status %s", status)
	}
	result, err := tx.Exec(`UPDATE mall_inventories SET reserved=reserved+?,sold=sold-?,version=version+1,updated_at=NOW() WHERE product_id=? AND sold>=?`, quantity, quantity, productID, quantity)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("inventory invariant violation while unconfirming")
	}
	_, err = tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=? AND status=?`, ReservationReserved, p.OrderID, ReservationConfirmed)
	return err
}

func lockReservation(tx *sql.Tx, orderID string) (productID, quantity int64, status string, err error) {
	err = tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, orderID).Scan(&productID, &quantity, &status)
	return
}

func ApplyPointsMutation(tx *sql.Tx, p SagaPayload, operation PointsOperation) error {
	switch operation {
	case PointsDebit:
		return debitPointsTx(tx, p)
	case PointsRefund:
		return refundPointsTx(tx, p)
	default:
		return fmt.Errorf("unknown points operation %q", operation)
	}
}

func debitPointsTx(tx *sql.Tx, p SagaPayload) error {
	if p.OrderID == "" || p.UserID <= 0 || p.TotalPoints <= 0 {
		return errors.New("invalid points mutation")
	}
	insert, err := tx.Exec(`INSERT IGNORE INTO mall_points_ledgers(order_id,operation,user_id,delta,created_at) VALUES(?,'debit',?,?,NOW())`, p.OrderID, p.UserID, -p.TotalPoints)
	if err != nil {
		return err
	}
	if affected, _ := insert.RowsAffected(); affected == 0 {
		var userID, delta int64
		if err = tx.QueryRow(`SELECT user_id,delta FROM mall_points_ledgers WHERE order_id=? AND operation='debit' FOR UPDATE`, p.OrderID).Scan(&userID, &delta); err != nil {
			return err
		}
		if userID != p.UserID || delta != -p.TotalPoints {
			return ErrIdempotencyConflict
		}
		return nil
	}
	result, err := tx.Exec(`UPDATE mall_points_accounts SET balance=balance-?,version=version+1,updated_at=NOW() WHERE user_id=? AND balance>=?`, p.TotalPoints, p.UserID, p.TotalPoints)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrInsufficientPoints
	}
	return nil
}

func refundPointsTx(tx *sql.Tx, p SagaPayload) error {
	var debitUserID, debit int64
	if err := tx.QueryRow(`SELECT user_id,delta FROM mall_points_ledgers WHERE order_id=? AND operation='debit' FOR UPDATE`, p.OrderID).Scan(&debitUserID, &debit); errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if debitUserID != p.UserID || debit >= 0 || -debit != p.TotalPoints {
		return ErrIdempotencyConflict
	}
	amount := -debit
	insert, err := tx.Exec(`INSERT IGNORE INTO mall_points_ledgers(order_id,operation,user_id,delta,created_at) VALUES(?,'refund',?,?,NOW())`, p.OrderID, p.UserID, amount)
	if err != nil {
		return err
	}
	if affected, _ := insert.RowsAffected(); affected == 0 {
		var refundUserID, refund int64
		if err = tx.QueryRow(`SELECT user_id,delta FROM mall_points_ledgers WHERE order_id=? AND operation='refund' FOR UPDATE`, p.OrderID).Scan(&refundUserID, &refund); err != nil {
			return err
		}
		if refundUserID != p.UserID || refund != amount {
			return ErrIdempotencyConflict
		}
		return nil
	}
	result, err := tx.Exec(`UPDATE mall_points_accounts SET balance=balance+?,version=version+1,updated_at=NOW() WHERE user_id=?`, amount, p.UserID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return errors.New("points account missing while refunding")
	}
	return nil
}
