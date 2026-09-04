package domain

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dtm-labs/client/dtmcli"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

var (
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInsufficientPoints  = errors.New("insufficient points")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
	ErrInjectedFailure     = errors.New("injected branch failure")
)

type BranchServer struct {
	DB      *sql.DB
	Role    string
	Metrics *BranchMetrics
}

func (s *BranchServer) Handler() http.Handler {
	if s.Metrics == nil {
		s.Metrics = NewBranchMetrics(s.Role)
	}
	mux := http.NewServeMux()
	switch s.Role {
	case "order":
		mux.HandleFunc("/dtm/order/prepare", s.branch("order_prepare", s.prepareOrder))
		mux.HandleFunc("/dtm/order/cancel", s.branch("order_cancel", s.cancelOrder))
		mux.HandleFunc("/dtm/order/confirm", s.branch("order_confirm", s.confirmOrder))
	case "inventory":
		mux.HandleFunc("/dtm/inventory/reserve", s.branch("inventory_reserve", s.reserveInventory))
		mux.HandleFunc("/dtm/inventory/release", s.branch("inventory_release", s.releaseInventory))
		mux.HandleFunc("/dtm/inventory/confirm", s.branch("inventory_confirm", s.confirmInventory))
		mux.HandleFunc("/dtm/inventory/unconfirm", s.branch("inventory_unconfirm", s.unconfirmInventory))
	case "points":
		mux.HandleFunc("/dtm/points/debit", s.branch("points_debit", s.debitPoints))
		mux.HandleFunc("/dtm/points/refund", s.branch("points_refund", s.refundPoints))
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("/metrics", s.Metrics.Handler())
	return mux
}

type branchOperation func(*sql.Tx, SagaPayload) error

func (s *BranchServer) branch(name string, operation branchOperation) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		result := "failure"
		defer func() {
			s.Metrics.Requests.WithLabelValues(name, result).Inc()
			s.Metrics.Latency.WithLabelValues(name).Observe(time.Since(started).Seconds())
		}()
		var payload SagaPayload
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
			writeBranchResult(w, fmt.Errorf("decode payload: %w", err))
			return
		}
		if payload.FailAt == name {
			writeBranchResult(w, fmt.Errorf("%w at %s", ErrInjectedFailure, name))
			return
		}
		var barrier *dtmcli.BranchBarrier
		var err error
		maxAttempts := branchMaxAttempts()
		for attempt := 0; attempt < maxAttempts; attempt++ {
			barrier, err = dtmcli.BarrierFromQuery(r.URL.Query())
			if err != nil {
				break
			}
			barrier.DBType = dtmcli.DBTypeMysql
			err = barrier.CallWithDB(s.DB, func(tx *sql.Tx) error { return operation(tx, payload) })
			if !isMySQLDeadlock(err) {
				break
			}
			s.Metrics.Retries.WithLabelValues(name, "mysql_deadlock").Inc()
			time.Sleep(time.Duration(attempt+1) * time.Millisecond)
		}
		if err == nil {
			result = "success"
		}
		if err != nil {
			gid, branchID := "", ""
			if barrier != nil {
				gid, branchID = barrier.Gid, barrier.BranchID
			}
			log.Printf("mall DTM branch role=%s operation=%s gid=%s branch=%s failed: %v", s.Role, name, gid, branchID, err)
		}
		writeBranchResult(w, err)
	}
}

func branchMaxAttempts() int {
	value, err := strconv.Atoi(os.Getenv("LIVECLASS_MALL_BRANCH_MAX_ATTEMPTS"))
	if err == nil && value >= 1 && value <= 10 {
		return value
	}
	return 3
}

func isMySQLDeadlock(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1213
}

func writeBranchResult(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(dtmcli.ResultSuccess))
		return
	}
	if errors.Is(err, ErrInsufficientStock) || errors.Is(err, ErrInsufficientPoints) || errors.Is(err, ErrIdempotencyConflict) || errors.Is(err, ErrInjectedFailure) {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, _ = w.Write([]byte(dtmcli.ResultFailure))
}

func (s *BranchServer) prepareOrder(tx *sql.Tx, p SagaPayload) error {
	if err := validatePayload(p); err != nil {
		return err
	}
	result, err := tx.Exec(`INSERT IGNORE INTO mall_orders
(id,user_id,request_id,product_id,quantity,total_points,status,saga_gid,created_at,updated_at)
VALUES(?,?,?,?,?,?,?, ?,NOW(),NOW())`, p.OrderID, p.UserID, p.RequestID, p.ProductID, p.Quantity, p.TotalPoints, OrderPending, p.SagaGID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		return nil
	}
	var orderID string
	var productID, quantity, total int64
	err = tx.QueryRow(`SELECT id,product_id,quantity,total_points FROM mall_orders WHERE user_id=? AND request_id=? FOR UPDATE`, p.UserID, p.RequestID).Scan(&orderID, &productID, &quantity, &total)
	if err != nil {
		return err
	}
	if orderID != p.OrderID || productID != p.ProductID || quantity != p.Quantity || total != p.TotalPoints {
		return ErrIdempotencyConflict
	}
	return nil
}

func (s *BranchServer) cancelOrder(tx *sql.Tx, p SagaPayload) error {
	_, err := tx.Exec(`UPDATE mall_orders SET status=?,updated_at=NOW() WHERE id=? AND status<>?`, OrderCompensated, p.OrderID, OrderCompensated)
	return err
}

func (s *BranchServer) confirmOrder(tx *sql.Tx, p SagaPayload) error {
	result, err := tx.Exec(`UPDATE mall_orders SET status=?,updated_at=NOW() WHERE id=? AND status=?`, OrderConfirmed, p.OrderID, OrderPending)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		return nil
	}
	var status string
	if err = tx.QueryRow(`SELECT status FROM mall_orders WHERE id=? FOR UPDATE`, p.OrderID).Scan(&status); err != nil {
		return err
	}
	if status != OrderConfirmed {
		return fmt.Errorf("cannot confirm order in status %s", status)
	}
	return nil
}

func (s *BranchServer) reserveInventory(tx *sql.Tx, p SagaPayload) error {
	if err := validatePayload(p); err != nil {
		return err
	}
	var productID, quantity int64
	var status string
	err := tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, p.OrderID).Scan(&productID, &quantity, &status)
	if err == nil {
		if productID != p.ProductID || quantity != p.Quantity {
			return ErrIdempotencyConflict
		}
		if status == ReservationReserved || status == ReservationConfirmed {
			return nil
		}
		return fmt.Errorf("reservation already %s", status)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	result, err := tx.Exec(`UPDATE mall_inventories SET available=available-?,reserved=reserved+?,version=version+1,updated_at=NOW() WHERE product_id=? AND available>=?`, p.Quantity, p.Quantity, p.ProductID, p.Quantity)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrInsufficientStock
	}
	_, err = tx.Exec(`INSERT INTO mall_inventory_reservations(order_id,product_id,quantity,status,created_at,updated_at) VALUES(?,?,?,?,NOW(),NOW())`, p.OrderID, p.ProductID, p.Quantity, ReservationReserved)
	return err
}

func (s *BranchServer) releaseInventory(tx *sql.Tx, p SagaPayload) error {
	var productID, quantity int64
	var status string
	err := tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, p.OrderID).Scan(&productID, &quantity, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil || status == ReservationReleased {
		return err
	}
	if status != ReservationReserved {
		return fmt.Errorf("cannot release reservation in status %s", status)
	}
	if _, err = tx.Exec(`UPDATE mall_inventories SET available=available+?,reserved=reserved-?,version=version+1,updated_at=NOW() WHERE product_id=? AND reserved>=?`, quantity, quantity, productID, quantity); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=?`, ReservationReleased, p.OrderID)
	return err
}

func (s *BranchServer) confirmInventory(tx *sql.Tx, p SagaPayload) error {
	var productID, quantity int64
	var status string
	if err := tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, p.OrderID).Scan(&productID, &quantity, &status); err != nil {
		return err
	}
	if status == ReservationConfirmed {
		return nil
	}
	if status != ReservationReserved {
		return fmt.Errorf("cannot confirm reservation in status %s", status)
	}
	if _, err := tx.Exec(`UPDATE mall_inventories SET reserved=reserved-?,sold=sold+?,version=version+1,updated_at=NOW() WHERE product_id=? AND reserved>=?`, quantity, quantity, productID, quantity); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=?`, ReservationConfirmed, p.OrderID)
	return err
}

func (s *BranchServer) unconfirmInventory(tx *sql.Tx, p SagaPayload) error {
	var productID, quantity int64
	var status string
	err := tx.QueryRow(`SELECT product_id,quantity,status FROM mall_inventory_reservations WHERE order_id=? FOR UPDATE`, p.OrderID).Scan(&productID, &quantity, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil || status == ReservationReserved || status == ReservationReleased {
		return err
	}
	if status != ReservationConfirmed {
		return fmt.Errorf("cannot unconfirm reservation in status %s", status)
	}
	if _, err = tx.Exec(`UPDATE mall_inventories SET reserved=reserved+?,sold=sold-?,version=version+1,updated_at=NOW() WHERE product_id=? AND sold>=?`, quantity, quantity, productID, quantity); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE mall_inventory_reservations SET status=?,updated_at=NOW() WHERE order_id=?`, ReservationReserved, p.OrderID)
	return err
}

func (s *BranchServer) debitPoints(tx *sql.Tx, p SagaPayload) error {
	if err := validatePayload(p); err != nil {
		return err
	}
	var delta int64
	err := tx.QueryRow(`SELECT delta FROM mall_points_ledgers WHERE order_id=? AND operation='debit' FOR UPDATE`, p.OrderID).Scan(&delta)
	if err == nil {
		if delta != -p.TotalPoints {
			return ErrIdempotencyConflict
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	result, err := tx.Exec(`UPDATE mall_points_accounts SET balance=balance-?,version=version+1,updated_at=NOW() WHERE user_id=? AND balance>=?`, p.TotalPoints, p.UserID, p.TotalPoints)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrInsufficientPoints
	}
	_, err = tx.Exec(`INSERT INTO mall_points_ledgers(order_id,operation,user_id,delta,created_at) VALUES(?,'debit',?,?,NOW())`, p.OrderID, p.UserID, -p.TotalPoints)
	return err
}

func (s *BranchServer) refundPoints(tx *sql.Tx, p SagaPayload) error {
	var debit int64
	if err := tx.QueryRow(`SELECT delta FROM mall_points_ledgers WHERE order_id=? AND operation='debit' FOR UPDATE`, p.OrderID).Scan(&debit); errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	var refund int64
	err := tx.QueryRow(`SELECT delta FROM mall_points_ledgers WHERE order_id=? AND operation='refund' FOR UPDATE`, p.OrderID).Scan(&refund)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	amount := -debit
	if _, err = tx.Exec(`UPDATE mall_points_accounts SET balance=balance+?,version=version+1,updated_at=NOW() WHERE user_id=?`, amount, p.UserID); err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO mall_points_ledgers(order_id,operation,user_id,delta,created_at) VALUES(?,'refund',?,?,NOW())`, p.OrderID, p.UserID, amount)
	return err
}

func validatePayload(p SagaPayload) error {
	if strings.TrimSpace(p.OrderID) == "" || p.UserID <= 0 || p.ProductID <= 0 || p.Quantity <= 0 || p.TotalPoints <= 0 || strings.TrimSpace(p.RequestID) == "" {
		return errors.New("invalid saga payload")
	}
	return nil
}
