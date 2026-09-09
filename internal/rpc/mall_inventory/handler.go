package main

import (
	"context"
	"database/sql"
	"errors"

	"liveclass/idl/kitex_gen/common"
	inventory "liveclass/idl/kitex_gen/mall_inventory"
	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm"
)

type inventoryServiceImpl struct {
	db  *gorm.DB
	raw *sql.DB
}

func inventoryResp(item *domain.Inventory) *inventory.Inventory {
	if item == nil {
		return nil
	}
	return &inventory.Inventory{ProductId: item.ProductID, Available: item.Available, Reserved: item.Reserved, Sold: item.Sold, Version: item.Version}
}

func (s *inventoryServiceImpl) GetInventory(ctx context.Context, req *inventory.GetInventoryReq) (*inventory.GetInventoryResp, error) {
	if req == nil || req.ProductId <= 0 {
		return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 400, Msg: "invalid product_id"}}, nil
	}
	item, err := s.load(ctx, req.ProductId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 404, Msg: "inventory not found"}}, nil
	}
	if err != nil {
		return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 500, Msg: "failed to query inventory"}}, nil
	}
	return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Inventory: inventoryResp(item)}, nil
}

func (s *inventoryServiceImpl) CheckSaleable(ctx context.Context, req *inventory.CheckSaleableReq) (*inventory.CheckSaleableResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	item, err := s.load(ctx, req.ProductId)
	if err != nil {
		return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 404, Msg: "inventory not found"}}, nil
	}
	// This is only a hint for UI display. Correctness remains in Reserve's
	// conditional UPDATE because availability can change immediately afterward.
	return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Saleable: item.Available >= req.Quantity, Available: item.Available}, nil
}

func (s *inventoryServiceImpl) Reserve(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, domain.InventoryReserve)
}

func (s *inventoryServiceImpl) Release(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, domain.InventoryRelease)
}

func (s *inventoryServiceImpl) Confirm(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, domain.InventoryConfirm)
}

func (s *inventoryServiceImpl) mutate(ctx context.Context, req *inventory.InventoryMutationReq, operation domain.InventoryOperation) (*inventory.InventoryMutationResp, error) {
	if req == nil || req.OrderId == "" || req.ProductId <= 0 || req.Quantity <= 0 {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.raw == nil || s.db == nil {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 500, Msg: "service not initialized"}}, nil
	}
	payload := domain.SagaPayload{OrderID: req.OrderId, ProductID: req.ProductId, Quantity: req.Quantity}
	if err := domain.RunLocalMutation(ctx, s.raw, func(tx *sql.Tx) error {
		return domain.ApplyInventoryMutation(tx, payload, operation)
	}); err != nil {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: mutationCode(err), Msg: err.Error()}}, nil
	}
	item, err := s.load(ctx, req.ProductId)
	if err != nil {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 500, Msg: "mutation committed but reload failed"}}, nil
	}
	return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Inventory: inventoryResp(item)}, nil
}

func (s *inventoryServiceImpl) load(ctx context.Context, productID int64) (*domain.Inventory, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("inventory service is not initialized")
	}
	var item domain.Inventory
	if err := s.db.WithContext(ctx).First(&item, "product_id = ?", productID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func mutationCode(err error) int16 {
	if errors.Is(err, domain.ErrInsufficientStock) || errors.Is(err, domain.ErrIdempotencyConflict) || errors.Is(err, gorm.ErrRecordNotFound) {
		return 409
	}
	return 500
}
