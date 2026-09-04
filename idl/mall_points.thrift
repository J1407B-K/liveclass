namespace go mall_points

include "common.thrift"

struct PointsAccount {
    1: i64 user_id,
    2: i64 balance,
    3: i64 version,
}

struct GetPointsReq {
    1: i64 user_id,
}

struct GetPointsResp {
    1: common.resp resp,
    2: optional PointsAccount account,
}

struct PointsMutationReq { 1: string order_id, 2: i64 user_id, 3: i64 amount }
struct PointsMutationResp { 1: common.resp resp, 2: optional PointsAccount account }

service PointsService {
    GetPointsResp GetPoints(1: GetPointsReq req),
    PointsMutationResp Debit(1: PointsMutationReq req),
    PointsMutationResp Refund(1: PointsMutationReq req),
}
