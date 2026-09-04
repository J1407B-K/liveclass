package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type ExchangeInput struct {
	UserID    int64
	RequestID string
	ProductID int64
	Quantity  int64
	FailAt    string
}

type SagaSubmitter interface {
	Submit(context.Context, SagaPayload) error
}

type Coordinator struct {
	DB       *gorm.DB
	Saga     SagaSubmitter
	requests singleflight.Group
}

func (c *Coordinator) Exchange(ctx context.Context, input ExchangeInput) (*Order, error) {
	input.RequestID = strings.TrimSpace(input.RequestID)
	if c == nil || c.DB == nil || c.Saga == nil {
		return nil, errors.New("mall coordinator is not initialized")
	}
	if input.UserID <= 0 || input.ProductID <= 0 || input.Quantity <= 0 || input.RequestID == "" || len(input.RequestID) > 64 {
		return nil, errors.New("invalid exchange request")
	}
	key := fmt.Sprintf("%d:%s", input.UserID, input.RequestID)
	result, err, _ := c.requests.Do(key, func() (any, error) {
		return c.exchange(ctx, input)
	})
	if err != nil {
		return nil, err
	}
	order, ok := result.(*Order)
	if !ok {
		return nil, errors.New("unexpected mall exchange result")
	}
	return order, nil
}

func (c *Coordinator) exchange(ctx context.Context, input ExchangeInput) (*Order, error) {

	var existing Order
	err := c.DB.WithContext(ctx).Where("user_id = ? AND request_id = ?", input.UserID, input.RequestID).First(&existing).Error
	if err == nil {
		if existing.ProductID != input.ProductID || int64(existing.Quantity) != input.Quantity {
			return nil, ErrIdempotencyConflict
		}
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var product Product
	if err = c.DB.WithContext(ctx).Where("id = ? AND active = ?", input.ProductID, true).First(&product).Error; err != nil {
		return nil, err
	}
	if product.PointsPrice <= 0 || input.Quantity > 100 || product.PointsPrice > (1<<62)/input.Quantity {
		return nil, errors.New("invalid product price or quantity")
	}
	orderID := deterministicOrderID(input.UserID, input.RequestID)
	payload := SagaPayload{
		OrderID: orderID, UserID: input.UserID, RequestID: input.RequestID,
		ProductID: input.ProductID, Quantity: input.Quantity, PointsPrice: product.PointsPrice,
		TotalPoints: product.PointsPrice * input.Quantity, SagaGID: "mall-" + orderID, FailAt: input.FailAt,
	}
	if err = c.Saga.Submit(ctx, payload); err != nil {
		var afterFailure Order
		if queryErr := c.DB.WithContext(ctx).Where("id = ?", orderID).First(&afterFailure).Error; queryErr == nil {
			if afterFailure.Status == OrderConfirmed {
				return &afterFailure, nil
			}
			return &afterFailure, err
		}
		return nil, err
	}
	var order Order
	if err = c.DB.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func deterministicOrderID(userID int64, requestID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("liveclass-mall:%d:%s", userID, requestID))).String()
}

type DTMSagaSubmitter struct {
	ServerURL    string
	OrderURL     string
	InventoryURL string
	PointsURL    string
	Timeout      time.Duration
}

func (s DTMSagaSubmitter) Submit(ctx context.Context, payload SagaPayload) error {
	if s.ServerURL == "" || s.OrderURL == "" || s.InventoryURL == "" || s.PointsURL == "" {
		return errors.New("incomplete DTM Saga configuration")
	}
	saga := dtmcli.NewSaga(strings.TrimRight(s.ServerURL, "/"), payload.SagaGID)
	saga.Context = ctx
	saga.WaitResult = true
	saga.RetryInterval = 1
	saga.RetryLimit = 3
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	saga.RequestTimeout = int64(timeout.Seconds())
	saga.Add(joinURL(s.OrderURL, "/dtm/order/prepare"), joinURL(s.OrderURL, "/dtm/order/cancel"), payload)
	saga.Add(joinURL(s.InventoryURL, "/dtm/inventory/reserve"), joinURL(s.InventoryURL, "/dtm/inventory/release"), payload)
	saga.Add(joinURL(s.PointsURL, "/dtm/points/debit"), joinURL(s.PointsURL, "/dtm/points/refund"), payload)
	saga.Add(joinURL(s.InventoryURL, "/dtm/inventory/confirm"), joinURL(s.InventoryURL, "/dtm/inventory/unconfirm"), payload)
	saga.Add(joinURL(s.OrderURL, "/dtm/order/confirm"), joinURL(s.OrderURL, "/dtm/order/cancel"), payload)
	return saga.Submit()
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
