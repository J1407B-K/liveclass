package service

import (
	"context"
	"net/http"
	"strings"

	"liveclass/idl/kitex_gen/mall"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"

	"github.com/cloudwego/hertz/pkg/app"
)

type mallExchangeRequest struct {
	RequestID string `json:"request_id"`
	ProductID int64  `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

func ListMallProducts(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	resp, err := global.Clients.MallClient.ListProducts(c, &mall.ListProductsReq{UserId: uid})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, model2.Response{Code: code.RPCError, Msg: "mall service unavailable"})
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func ExchangeMallProduct(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	var request mallExchangeRequest
	if err := ctx.BindJSON(&request); err != nil || strings.TrimSpace(request.RequestID) == "" || request.ProductID <= 0 || request.Quantity <= 0 {
		ctx.JSON(http.StatusBadRequest, model2.Response{Code: code.BadRequest, Msg: "request_id, product_id and positive quantity are required"})
		return
	}
	resp, err := global.Clients.MallClient.Exchange(c, &mall.ExchangeReq{UserId: uid, RequestId: request.RequestID, ProductId: request.ProductID, Quantity: request.Quantity})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, model2.Response{Code: code.RPCError, Msg: "mall service unavailable"})
		return
	}
	status := http.StatusOK
	if resp.Resp != nil && resp.Resp.Code != 0 {
		status = http.StatusConflict
	}
	ctx.JSON(status, resp)
}

func GetMallOrder(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	orderID := strings.TrimSpace(ctx.Param("order_id"))
	if orderID == "" {
		ctx.JSON(http.StatusBadRequest, model2.Response{Code: code.BadRequest, Msg: "order_id is required"})
		return
	}
	resp, err := global.Clients.MallClient.GetOrder(c, &mall.GetOrderReq{UserId: uid, OrderId: orderID})
	if err != nil {
		ctx.JSON(http.StatusBadGateway, model2.Response{Code: code.RPCError, Msg: "mall service unavailable"})
		return
	}
	status := http.StatusOK
	if resp.Resp != nil && resp.Resp.Code != 0 {
		status = http.StatusNotFound
	}
	ctx.JSON(status, resp)
}
