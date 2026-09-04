package main

import (
	"context"

	"liveclass/idl/kitex_gen/common"
	inventory "liveclass/idl/kitex_gen/mall_inventory"
	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm"
)

type inventoryServiceImpl struct{ db *gorm.DB }

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
	if s == nil || s.db == nil {
		return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 500, Msg: "inventory service is not initialized"}}, nil
	}
	var item domain.Inventory
	if err := s.db.WithContext(ctx).First(&item, "product_id = ?", req.ProductId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 404, Msg: "inventory not found"}}, nil
		}
		return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 500, Msg: "failed to query inventory"}}, nil
	}
	return &inventory.GetInventoryResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Inventory: inventoryResp(&item)}, nil
}

func (s *inventoryServiceImpl) CheckSaleable(ctx context.Context, req *inventory.CheckSaleableReq) (*inventory.CheckSaleableResp, error) {
	if req == nil || req.ProductId <= 0 || req.Quantity <= 0 {
		return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	var item domain.Inventory
	if err := s.db.WithContext(ctx).First(&item, "product_id = ?", req.ProductId).Error; err != nil {
		return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 404, Msg: "inventory not found"}}, nil
	}
	return &inventory.CheckSaleableResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Saleable: item.Available >= req.Quantity, Available: item.Available}, nil
}

func (s *inventoryServiceImpl) Reserve(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, "reserve")
}
func (s *inventoryServiceImpl) Release(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, "release")
}
func (s *inventoryServiceImpl) Confirm(ctx context.Context, req *inventory.InventoryMutationReq) (*inventory.InventoryMutationResp, error) {
	return s.mutate(ctx, req, "confirm")
}
func (s *inventoryServiceImpl) mutate(ctx context.Context, req *inventory.InventoryMutationReq, op string) (*inventory.InventoryMutationResp, error) {
	if req == nil || req.OrderId == "" || req.ProductId <= 0 || req.Quantity <= 0 {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.db == nil {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 500, Msg: "service not initialized"}}, nil
	}
	var item domain.Inventory
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&item, "product_id = ?", req.ProductId).Error; err != nil {
			return err
		}
		var r domain.InventoryReservation
		found := tx.First(&r, "order_id = ?", req.OrderId).Error == nil
		if op == "reserve" && !found {
			if item.Available < req.Quantity {
				return domain.ErrInsufficientStock
			}
			item.Available -= req.Quantity
			item.Reserved += req.Quantity
			r = domain.InventoryReservation{OrderID: req.OrderId, ProductID: req.ProductId, Quantity: req.Quantity, Status: domain.ReservationReserved}
			if err := tx.Save(&item).Error; err != nil {
				return err
			}
			return tx.Create(&r).Error
		}
		if !found {
			return gorm.ErrRecordNotFound
		}
		if op == "release" && r.Status == domain.ReservationReserved {
			item.Available += r.Quantity
			item.Reserved -= r.Quantity
			r.Status = domain.ReservationReleased
		}
		if op == "confirm" && r.Status == domain.ReservationReserved {
			item.Reserved -= r.Quantity
			item.Sold += r.Quantity
			r.Status = domain.ReservationConfirmed
		}
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		return tx.Save(&r).Error
	})
	if err != nil {
		return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 409, Msg: err.Error()}}, nil
	}
	return &inventory.InventoryMutationResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Inventory: inventoryResp(&item)}, nil
}
