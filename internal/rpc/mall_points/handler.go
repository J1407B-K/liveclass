package main

import (
	"context"

	"liveclass/idl/kitex_gen/common"
	points "liveclass/idl/kitex_gen/mall_points"
	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm"
)

type pointsServiceImpl struct{ db *gorm.DB }

func pointsResp(a *domain.PointsAccount) *points.PointsAccount {
	if a == nil {
		return nil
	}
	return &points.PointsAccount{UserId: a.UserID, Balance: a.Balance, Version: a.Version}
}

func (s *pointsServiceImpl) GetPoints(ctx context.Context, req *points.GetPointsReq) (*points.GetPointsResp, error) {
	if req == nil || req.UserId <= 0 {
		return &points.GetPointsResp{Resp: &common.Resp{Code: 400, Msg: "invalid user_id"}}, nil
	}
	if s == nil || s.db == nil {
		return &points.GetPointsResp{Resp: &common.Resp{Code: 500, Msg: "points service is not initialized"}}, nil
	}
	var account domain.PointsAccount
	if err := s.db.WithContext(ctx).First(&account, "user_id = ?", req.UserId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &points.GetPointsResp{Resp: &common.Resp{Code: 404, Msg: "points account not found"}}, nil
		}
		return &points.GetPointsResp{Resp: &common.Resp{Code: 500, Msg: "failed to query points account"}}, nil
	}
	return &points.GetPointsResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Account: pointsResp(&account)}, nil
}

func (s *pointsServiceImpl) Debit(ctx context.Context, req *points.PointsMutationReq) (*points.PointsMutationResp, error) {
	return s.mutate(ctx, req, false)
}
func (s *pointsServiceImpl) Refund(ctx context.Context, req *points.PointsMutationReq) (*points.PointsMutationResp, error) {
	return s.mutate(ctx, req, true)
}
func (s *pointsServiceImpl) mutate(ctx context.Context, req *points.PointsMutationReq, refund bool) (*points.PointsMutationResp, error) {
	if req == nil || req.OrderId == "" || req.UserId <= 0 || req.Amount <= 0 {
		return &points.PointsMutationResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	var account domain.PointsAccount
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&account, "user_id = ?", req.UserId).Error; err != nil {
			return err
		}
		op := "debit"
		delta := -req.Amount
		if refund {
			op = "refund"
			delta = req.Amount
		}
		var l domain.PointsLedger
		if tx.Where("order_id = ? AND operation = ?", req.OrderId, op).First(&l).Error == nil {
			return nil
		}
		if !refund && account.Balance < req.Amount {
			return domain.ErrInsufficientPoints
		}
		account.Balance += delta
		if err := tx.Save(&account).Error; err != nil {
			return err
		}
		return tx.Create(&domain.PointsLedger{OrderID: req.OrderId, Operation: op, UserID: req.UserId, Delta: delta}).Error
	})
	if err != nil {
		return &points.PointsMutationResp{Resp: &common.Resp{Code: 409, Msg: err.Error()}}, nil
	}
	return &points.PointsMutationResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Account: pointsResp(&account)}, nil
}
