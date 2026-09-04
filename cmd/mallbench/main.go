package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm/clause"
)

type benchmarkResult struct {
	GeneratedAt   string             `json:"generated_at"`
	Scenario      string             `json:"scenario"`
	Environment   map[string]any     `json:"environment"`
	Workload      map[string]any     `json:"workload"`
	LatencyMS     map[string]float64 `json:"latency_ms"`
	ThroughputQPS float64            `json:"throughput_qps"`
	DurationMS    float64            `json:"duration_ms"`
	Calls         int                `json:"calls"`
	Confirmed     int64              `json:"confirmed_orders"`
	Compensated   int64              `json:"compensated_orders"`
	CallErrors    int64              `json:"call_errors"`
	FinalState    map[string]int64   `json:"final_state"`
	Assertions    map[string]bool    `json:"assertions"`
	ClientRuntime map[string]int64   `json:"client_runtime"`
}

func main() {
	var scenario, output, dtmServer, runID string
	var requests, concurrency int
	var stock, initialPoints, price int64
	flag.StringVar(&scenario, "scenario", "oversell", "normal|oversell|dedup|compensation")
	flag.StringVar(&output, "output", "", "result JSON path")
	flag.StringVar(&dtmServer, "dtm", "http://127.0.0.1:36789/api/dtmsvr", "DTM HTTP endpoint")
	flag.StringVar(&runID, "run-id", fmt.Sprintf("%x", time.Now().UnixNano()), "unique run id")
	flag.IntVar(&requests, "requests", 100, "number of exchange calls")
	flag.IntVar(&concurrency, "concurrency", 20, "worker count")
	flag.Int64Var(&stock, "stock", 50, "initial stock")
	flag.Int64Var(&initialPoints, "points", 1000, "initial points per user")
	flag.Int64Var(&price, "price", 100, "points price")
	flag.Parse()
	if requests <= 0 || concurrency <= 0 || stock < 0 || initialPoints < 0 || price <= 0 {
		panic("invalid benchmark arguments")
	}

	db, rawDB, err := domain.OpenMySQL()
	must(err)
	must(domain.EnsureBarrierTable(rawDB))
	must(domain.MigrateOrder(db))
	must(domain.MigrateInventory(db))
	must(domain.MigratePoints(db))

	productID := int64(9_900_000_000) + time.Now().UnixNano()%1_000_000
	baseUserID := int64(8_000_000_000) + time.Now().UnixNano()%1_000_000
	product := domain.Product{ID: productID, Name: "Mall Benchmark Product", Description: scenario, PointsPrice: price, Active: true}
	must(db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&product).Error)
	must(db.Create(&domain.Inventory{ProductID: productID, Available: stock, Version: 1}).Error)
	users := requests
	if scenario == "dedup" {
		users = 1
	}
	accounts := make([]domain.PointsAccount, 0, users)
	for i := 0; i < users; i++ {
		accounts = append(accounts, domain.PointsAccount{UserID: baseUserID + int64(i), Balance: initialPoints, Version: 1})
	}
	must(db.CreateInBatches(accounts, 500).Error)

	coordinator := &domain.Coordinator{DB: db, Saga: domain.DTMSagaSubmitter{
		ServerURL:    dtmServer,
		OrderURL:     "http://host.docker.internal:19100",
		InventoryURL: "http://host.docker.internal:19101",
		PointsURL:    "http://host.docker.internal:19102",
		Timeout:      20 * time.Second,
	}}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	beforeGoroutines := runtime.NumGoroutine()
	jobs := make(chan int)
	latencies := make([]float64, requests)
	var callErrors atomic.Int64
	var wg sync.WaitGroup
	started := time.Now()
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				userID := baseUserID + int64(index)
				requestID := fmt.Sprintf("%s-%d", runID, index)
				if scenario == "dedup" {
					userID, requestID = baseUserID, runID+"-same"
				}
				failAt := ""
				if scenario == "compensation" {
					failAt = "points_debit"
				}
				callStarted := time.Now()
				_, exchangeErr := coordinator.Exchange(context.Background(), domain.ExchangeInput{UserID: userID, RequestID: requestID, ProductID: productID, Quantity: 1, FailAt: failAt})
				latencies[index] = float64(time.Since(callStarted).Microseconds()) / 1000
				if exchangeErr != nil {
					callErrors.Add(1)
				}
			}
		}()
	}
	for i := 0; i < requests; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	duration := time.Since(started)
	runtime.ReadMemStats(&after)

	var confirmed, compensated, reservations, sold, available, reserved, debitTotal, refundTotal, finalBalance int64
	db.Model(&domain.Order{}).Where("product_id = ? AND status = ?", productID, domain.OrderConfirmed).Count(&confirmed)
	db.Model(&domain.Order{}).Where("product_id = ? AND status = ?", productID, domain.OrderCompensated).Count(&compensated)
	db.Model(&domain.InventoryReservation{}).Where("product_id = ?", productID).Count(&reservations)
	db.Model(&domain.Inventory{}).Select("available").Where("product_id = ?", productID).Scan(&available)
	db.Model(&domain.Inventory{}).Select("reserved").Where("product_id = ?", productID).Scan(&reserved)
	db.Model(&domain.Inventory{}).Select("sold").Where("product_id = ?", productID).Scan(&sold)
	db.Model(&domain.PointsLedger{}).Where("user_id >= ? AND user_id < ? AND operation = 'debit'", baseUserID, baseUserID+int64(users)).Select("COALESCE(-SUM(delta),0)").Scan(&debitTotal)
	db.Model(&domain.PointsLedger{}).Where("user_id >= ? AND user_id < ? AND operation = 'refund'", baseUserID, baseUserID+int64(users)).Select("COALESCE(SUM(delta),0)").Scan(&refundTotal)
	db.Model(&domain.PointsAccount{}).Where("user_id >= ? AND user_id < ?", baseUserID, baseUserID+int64(users)).Select("COALESCE(SUM(balance),0)").Scan(&finalBalance)

	expectedConfirmed := int64(requests)
	if scenario == "oversell" && stock < expectedConfirmed {
		expectedConfirmed = stock
	}
	if scenario == "dedup" {
		expectedConfirmed = 1
	}
	if scenario == "compensation" {
		expectedConfirmed = 0
	}
	expectedBalance := int64(users)*initialPoints - expectedConfirmed*price
	expectedCompensated := int64(0)
	if scenario == "oversell" {
		expectedCompensated = int64(requests) - expectedConfirmed
	}
	if scenario == "compensation" {
		expectedCompensated = int64(requests)
	}
	result := benchmarkResult{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Scenario: scenario,
		Environment: map[string]any{"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "database": "MySQL 8 Docker", "coordinator": "DTM 1.19.0 boltdb Docker"},
		Workload:    map[string]any{"requests": requests, "concurrency": concurrency, "initial_stock": stock, "initial_points_per_user": initialPoints, "points_price": price},
		LatencyMS:   percentiles(latencies), ThroughputQPS: float64(requests) / duration.Seconds(), DurationMS: float64(duration.Microseconds()) / 1000,
		Calls: requests, Confirmed: confirmed, Compensated: compensated, CallErrors: callErrors.Load(),
		FinalState:    map[string]int64{"available": available, "reserved": reserved, "sold": sold, "reservations": reservations, "debited_points": debitTotal, "refunded_points": refundTotal, "total_user_balance": finalBalance},
		Assertions:    map[string]bool{"confirmed_matches_expected": confirmed == expectedConfirmed, "compensated_matches_expected": compensated == expectedCompensated, "no_oversell": sold <= stock && sold == confirmed, "no_reserved_leak": reserved == 0, "inventory_conserved": available+reserved+sold == stock, "points_conserved": finalBalance == expectedBalance, "compensation_refunds_match": debitTotal-refundTotal == confirmed*price},
		ClientRuntime: map[string]int64{"heap_alloc_delta_bytes": int64(after.HeapAlloc) - int64(before.HeapAlloc), "goroutines_before": int64(beforeGoroutines), "goroutines_after": int64(runtime.NumGoroutine())},
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	must(err)
	if output != "" {
		must(os.WriteFile(output, append(raw, '\n'), 0o644))
	}
	fmt.Println(string(raw))
	for name, passed := range result.Assertions {
		if !passed {
			panic("assertion failed: " + name)
		}
	}
}

func percentiles(values []float64) map[string]float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	pick := func(q float64) float64 {
		index := int(float64(len(sorted)-1) * q)
		return sorted[index]
	}
	return map[string]float64{"p50": pick(0.50), "p95": pick(0.95), "p99": pick(0.99), "max": sorted[len(sorted)-1]}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
