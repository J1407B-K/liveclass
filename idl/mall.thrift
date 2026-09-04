namespace go mall

include "common.thrift"

struct Product {
    1: i64 id,
    2: string name,
    3: string description,
    4: i64 points_price,
    5: bool active,
}

struct Order {
    1: string order_id,
    2: i64 user_id,
    3: string request_id,
    4: i64 product_id,
    5: i32 quantity,
    6: i64 total_points,
    7: string status,
    8: string saga_gid,
    9: i64 created_at_unix,
}

struct ExchangeReq {
    1: i64 user_id,
    2: string request_id,
    3: i64 product_id,
    4: i32 quantity,
}

struct ExchangeResp {
    1: common.resp resp,
    2: optional Order order,
}

struct GetOrderReq {
    1: i64 user_id,
    2: string order_id,
}

struct GetOrderResp {
    1: common.resp resp,
    2: optional Order order,
}

struct ListProductsReq {
    1: i64 user_id,
}

struct ListProductsResp {
    1: common.resp resp,
    2: list<Product> products,
}

service MallService {
    ExchangeResp Exchange(1: ExchangeReq req),
    GetOrderResp GetOrder(1: GetOrderReq req),
    ListProductsResp ListProducts(1: ListProductsReq req),
}
