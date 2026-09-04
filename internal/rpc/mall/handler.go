package main

import (
	"context"
	"errors"
	"strings"

	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/mall"
	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm"
)

type MallServiceImpl struct {
	coordinator *domain.Coordinator
	db          *gorm.DB
}

func (s *MallServiceImpl) Exchange(ctx context.Context, req *mall.ExchangeReq) (*mall.ExchangeResp, error) {
	if req == nil || req.UserId <= 0 || req.ProductId <= 0 || req.Quantity <= 0 || req.Quantity > 100 || strings.TrimSpace(req.RequestId) == "" {
		return &mall.ExchangeResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.coordinator == nil {
		return &mall.ExchangeResp{Resp: &common.Resp{Code: 500, Msg: "mall service is not initialized"}}, nil
	}
	order, err := s.coordinator.Exchange(ctx, domain.ExchangeInput{UserID: req.UserId, RequestID: req.RequestId, ProductID: req.ProductId, Quantity: int64(req.Quantity)})
	if err != nil {
		code := int16(500)
		if strings.HasPrefix(err.Error(), "invalid exchange request") || strings.HasPrefix(err.Error(), "invalid product") {
			code = 400
		}
		if errors.Is(err, domain.ErrInsufficientStock) || errors.Is(err, domain.ErrInsufficientPoints) || errors.Is(err, domain.ErrIdempotencyConflict) || errors.Is(err, gorm.ErrRecordNotFound) {
			code = 409
		}
		resp := &mall.ExchangeResp{Resp: &common.Resp{Code: code, Msg: err.Error()}}
		if order != nil {
			resp.Order = toRPCOrder(order)
		}
		return resp, nil
	}
	return &mall.ExchangeResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Order: toRPCOrder(order)}, nil
}

func (s *MallServiceImpl) GetOrder(ctx context.Context, req *mall.GetOrderReq) (*mall.GetOrderResp, error) {
	if req == nil || req.UserId <= 0 || req.OrderId == "" {
		return &mall.GetOrderResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.db == nil {
		return &mall.GetOrderResp{Resp: &common.Resp{Code: 500, Msg: "mall service is not initialized"}}, nil
	}
	var order domain.Order
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", req.OrderId, req.UserId).First(&order).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return &mall.GetOrderResp{Resp: &common.Resp{Code: 500, Msg: "failed to query order"}}, nil
		}
		return &mall.GetOrderResp{Resp: &common.Resp{Code: 404, Msg: "order not found"}}, nil
	}
	return &mall.GetOrderResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Order: toRPCOrder(&order)}, nil
}

func (s *MallServiceImpl) ListProducts(ctx context.Context, req *mall.ListProductsReq) (*mall.ListProductsResp, error) {
	if req == nil || req.UserId <= 0 {
		return &mall.ListProductsResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.db == nil {
		return &mall.ListProductsResp{Resp: &common.Resp{Code: 500, Msg: "mall service is not initialized"}}, nil
	}
	var products []domain.Product
	if err := s.db.WithContext(ctx).Where("active = ?", true).Order("id asc").Limit(100).Find(&products).Error; err != nil {
		return &mall.ListProductsResp{Resp: &common.Resp{Code: 500, Msg: "failed to query products"}}, nil
	}
	out := make([]*mall.Product, 0, len(products))
	for _, product := range products {
		out = append(out, &mall.Product{Id: product.ID, Name: product.Name, Description: product.Description, PointsPrice: product.PointsPrice, Active: product.Active})
	}
	return &mall.ListProductsResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Products: out}, nil
}

func toRPCOrder(order *domain.Order) *mall.Order {
	if order == nil {
		return nil
	}
	return &mall.Order{OrderId: order.ID, UserId: order.UserID, RequestId: order.RequestID, ProductId: order.ProductID, Quantity: order.Quantity, TotalPoints: order.TotalPoints, Status: order.Status, SagaGid: order.SagaGID, CreatedAtUnix: order.CreatedAt.Unix()}
}
