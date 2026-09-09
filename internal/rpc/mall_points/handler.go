package main

import (
	"context"
	"database/sql"
	"errors"

	"liveclass/idl/kitex_gen/common"
	points "liveclass/idl/kitex_gen/mall_points"
	"liveclass/internal/rpc/mall/domain"

	"gorm.io/gorm"
)

type pointsServiceImpl struct {
	db  *gorm.DB
	raw *sql.DB
}

func pointsResp(account *domain.PointsAccount) *points.PointsAccount {
	if account == nil {
		return nil
	}
	return &points.PointsAccount{UserId: account.UserID, Balance: account.Balance, Version: account.Version}
}

func (s *pointsServiceImpl) GetPoints(ctx context.Context, req *points.GetPointsReq) (*points.GetPointsResp, error) {
	if req == nil || req.UserId <= 0 {
		return &points.GetPointsResp{Resp: &common.Resp{Code: 400, Msg: "invalid user_id"}}, nil
	}
	account, err := s.load(ctx, req.UserId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &points.GetPointsResp{Resp: &common.Resp{Code: 404, Msg: "points account not found"}}, nil
	}
	if err != nil {
		return &points.GetPointsResp{Resp: &common.Resp{Code: 500, Msg: "failed to query points account"}}, nil
	}
	return &points.GetPointsResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Account: pointsResp(account)}, nil
}

func (s *pointsServiceImpl) Debit(ctx context.Context, req *points.PointsMutationReq) (*points.PointsMutationResp, error) {
	return s.mutate(ctx, req, domain.PointsDebit)
}

func (s *pointsServiceImpl) Refund(ctx context.Context, req *points.PointsMutationReq) (*points.PointsMutationResp, error) {
	return s.mutate(ctx, req, domain.PointsRefund)
}

func (s *pointsServiceImpl) mutate(ctx context.Context, req *points.PointsMutationReq, operation domain.PointsOperation) (*points.PointsMutationResp, error) {
	if req == nil || req.OrderId == "" || req.UserId <= 0 || req.Amount <= 0 {
		return &points.PointsMutationResp{Resp: &common.Resp{Code: 400, Msg: "invalid request"}}, nil
	}
	if s == nil || s.raw == nil || s.db == nil {
		return &points.PointsMutationResp{Resp: &common.Resp{Code: 500, Msg: "service not initialized"}}, nil
	}
	payload := domain.SagaPayload{OrderID: req.OrderId, UserID: req.UserId, TotalPoints: req.Amount}
	if err := domain.RunLocalMutation(ctx, s.raw, func(tx *sql.Tx) error {
		return domain.ApplyPointsMutation(tx, payload, operation)
	}); err != nil {
		code := int16(500)
		if errors.Is(err, domain.ErrInsufficientPoints) || errors.Is(err, domain.ErrIdempotencyConflict) {
			code = 409
		}
		return &points.PointsMutationResp{Resp: &common.Resp{Code: code, Msg: err.Error()}}, nil
	}
	account, err := s.load(ctx, req.UserId)
	if err != nil {
		return &points.PointsMutationResp{Resp: &common.Resp{Code: 500, Msg: "mutation committed but reload failed"}}, nil
	}
	return &points.PointsMutationResp{Resp: &common.Resp{Code: 0, Msg: "success"}, Account: pointsResp(account)}, nil
}

func (s *pointsServiceImpl) load(ctx context.Context, userID int64) (*domain.PointsAccount, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("points service is not initialized")
	}
	var account domain.PointsAccount
	if err := s.db.WithContext(ctx).First(&account, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
