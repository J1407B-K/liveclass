namespace go mall_inventory

include "common.thrift"

struct Inventory {
    1: i64 product_id,
    2: i64 available,
    3: i64 reserved,
    4: i64 sold,
    5: i64 version,
}

struct GetInventoryReq {
    1: i64 product_id,
}

struct GetInventoryResp {
    1: common.resp resp,
    2: optional Inventory inventory,
}

struct CheckSaleableReq { 1: i64 product_id, 2: i64 quantity }
struct CheckSaleableResp { 1: common.resp resp, 2: bool saleable, 3: i64 available }
struct InventoryMutationReq { 1: string order_id, 2: i64 product_id, 3: i64 quantity }
struct InventoryMutationResp { 1: common.resp resp, 2: optional Inventory inventory }

service InventoryService {
    GetInventoryResp GetInventory(1: GetInventoryReq req),
    CheckSaleableResp CheckSaleable(1: CheckSaleableReq req),
    InventoryMutationResp Reserve(1: InventoryMutationReq req),
    InventoryMutationResp Release(1: InventoryMutationReq req),
    InventoryMutationResp Confirm(1: InventoryMutationReq req),
}
