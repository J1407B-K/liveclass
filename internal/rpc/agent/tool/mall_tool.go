package tool

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/google/uuid"

	"liveclass/idl/kitex_gen/mall"
	"liveclass/idl/kitex_gen/mall/mallservice"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/toolruntime"
)

const mallConfirmationTTL = 5 * time.Minute

type MallCatalogRequest struct{}

type MallProduct struct {
	ProductID   int64  `json:"product_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PointsPrice int64  `json:"points_price"`
}

type MallCatalogResponse struct {
	Products []MallProduct `json:"products"`
}

type MallPrepareRequest struct {
	ProductID int64 `json:"product_id" jsonschema_description:"要兑换的商品ID"`
	Quantity  int32 `json:"quantity" jsonschema_description:"兑换数量，必须大于0"`
}

type MallPrepareResponse struct {
	Product           MallProduct `json:"product"`
	Quantity          int32       `json:"quantity"`
	TotalPoints       int64       `json:"total_points"`
	ConfirmationToken string      `json:"confirmation_token"`
	ExpiresAtUnix     int64       `json:"expires_at_unix"`
	Instruction       string      `json:"instruction"`
}

type MallExchangeRequest struct {
	ConfirmationToken string `json:"confirmation_token" jsonschema_description:"上一轮 prepare_mall_exchange 返回且经用户确认的令牌"`
}

type MallExchangeResponse struct {
	OrderID     string `json:"order_id"`
	ProductID   int64  `json:"product_id"`
	Quantity    int32  `json:"quantity"`
	TotalPoints int64  `json:"total_points"`
	Status      string `json:"status"`
}

type mallConfirmation struct {
	UserID          int64  `json:"user_id"`
	SessionID       string `json:"session_id"`
	IssuedRequestID string `json:"issued_request_id"`
	CheckoutID      string `json:"checkout_id"`
	ProductID       int64  `json:"product_id"`
	Quantity        int32  `json:"quantity"`
	ExpiresAtUnix   int64  `json:"expires_at_unix"`
}

func NewMallTools(cli mallservice.Client, secret []byte) (tool.InvokableTool, tool.InvokableTool, tool.InvokableTool, error) {
	if cli == nil {
		return nil, nil, nil, errors.New("nil mall service client")
	}
	if len(secret) < 32 {
		return nil, nil, nil, errors.New("mall confirmation secret must contain at least 32 bytes")
	}

	catalog, err := utils.InferTool("list_mall_products", "列出课内积分商城当前可兑换商品；这是只读操作", func(ctx context.Context, _ *MallCatalogRequest) (*MallCatalogResponse, error) {
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok {
			return nil, errors.New("missing authenticated principal")
		}
		return listMallProducts(ctx, cli, principal.UserID)
	})
	if err != nil {
		return nil, nil, nil, err
	}

	prepare, err := utils.InferTool("prepare_mall_exchange", "生成积分商品兑换报价和短期签名令牌；此步骤不会扣积分，必须询问用户确认", func(ctx context.Context, req *MallPrepareRequest) (*MallPrepareResponse, error) {
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok || req == nil || req.ProductID <= 0 || req.Quantity <= 0 || req.Quantity > 100 {
			return nil, errors.New("invalid mall exchange preview")
		}
		catalogResp, err := listMallProducts(ctx, cli, principal.UserID)
		if err != nil {
			return nil, err
		}
		var selected *MallProduct
		for i := range catalogResp.Products {
			if catalogResp.Products[i].ProductID == req.ProductID {
				selected = &catalogResp.Products[i]
				break
			}
		}
		if selected == nil {
			return nil, errors.New("product is not available")
		}
		expires := time.Now().Add(mallConfirmationTTL).Unix()
		claims := mallConfirmation{UserID: principal.UserID, SessionID: principal.SessionID, IssuedRequestID: principal.RequestID, CheckoutID: uuid.NewString(), ProductID: req.ProductID, Quantity: req.Quantity, ExpiresAtUnix: expires}
		token, err := signMallConfirmation(claims, secret)
		if err != nil {
			return nil, err
		}
		return &MallPrepareResponse{Product: *selected, Quantity: req.Quantity, TotalPoints: selected.PointsPrice * int64(req.Quantity), ConfirmationToken: token, ExpiresAtUnix: expires, Instruction: "本轮禁止兑换；向用户展示报价，下一轮收到明确确认后再提交此令牌"}, nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	exchange, err := utils.InferTool("exchange_mall_product", "使用上一轮的签名确认令牌执行积分兑换；会真实扣积分和库存", func(ctx context.Context, req *MallExchangeRequest) (*MallExchangeResponse, error) {
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok || req == nil || strings.TrimSpace(req.ConfirmationToken) == "" {
			return nil, errors.New("missing exchange confirmation")
		}
		claims, err := verifyMallConfirmation(req.ConfirmationToken, secret)
		if err != nil {
			return nil, err
		}
		if claims.UserID != principal.UserID || claims.SessionID != principal.SessionID {
			return nil, errors.New("confirmation does not belong to this user session")
		}
		if claims.IssuedRequestID == principal.RequestID {
			return nil, errors.New("exchange requires explicit confirmation in a later user turn")
		}
		resp, err := dependency.Do(ctx, dependency.InternalRPC, "mall_exchange", func(callCtx context.Context) (*mall.ExchangeResp, error) {
			return cli.Exchange(callCtx, &mall.ExchangeReq{UserId: principal.UserID, RequestId: claims.CheckoutID, ProductId: claims.ProductID, Quantity: claims.Quantity})
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.GetResp() == nil || resp.GetResp().GetCode() != 0 || resp.GetOrder() == nil {
			message := "mall exchange failed"
			if resp != nil && resp.GetResp() != nil {
				message = resp.GetResp().GetMsg()
			}
			return nil, errors.New(message)
		}
		order := resp.GetOrder()
		return &MallExchangeResponse{OrderID: order.GetOrderId(), ProductID: order.GetProductId(), Quantity: order.GetQuantity(), TotalPoints: order.GetTotalPoints(), Status: order.GetStatus()}, nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return catalog, prepare, exchange, nil
}

func listMallProducts(ctx context.Context, cli mallservice.Client, userID int64) (*MallCatalogResponse, error) {
	resp, err := dependency.Do(ctx, dependency.InternalRPC, "mall_list_products", func(callCtx context.Context) (*mall.ListProductsResp, error) {
		return cli.ListProducts(callCtx, &mall.ListProductsReq{UserId: userID})
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetResp() == nil || resp.GetResp().GetCode() != 0 {
		return nil, errors.New("mall catalog unavailable")
	}
	products := make([]MallProduct, 0, len(resp.GetProducts()))
	for _, product := range resp.GetProducts() {
		products = append(products, MallProduct{ProductID: product.GetId(), Name: product.GetName(), Description: product.GetDescription(), PointsPrice: product.GetPointsPrice()})
	}
	return &MallCatalogResponse{Products: products}, nil
}

func signMallConfirmation(claims mallConfirmation, secret []byte) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyMallConfirmation(token string, secret []byte) (mallConfirmation, error) {
	var claims mallConfirmation
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, errors.New("invalid confirmation token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errors.New("invalid confirmation signature")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return claims, errors.New("invalid confirmation signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(payload, &claims) != nil {
		return claims, errors.New("invalid confirmation payload")
	}
	if claims.UserID <= 0 || claims.ProductID <= 0 || claims.Quantity <= 0 || claims.CheckoutID == "" || claims.SessionID == "" || claims.IssuedRequestID == "" {
		return claims, errors.New("incomplete confirmation payload")
	}
	if time.Now().Unix() > claims.ExpiresAtUnix {
		return claims, fmt.Errorf("confirmation token expired")
	}
	return claims, nil
}
