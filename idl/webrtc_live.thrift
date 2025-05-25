namespace go webrtc_live

include "common.thrift"

struct BroadcastReq{
    1:string lesson_id,
    2:string b64offer,
}

struct BroadcastResp{
    1:common.resp resp,
}

struct ViewReq{
    1: string lesson_id,
    2: string b64offer,
}

struct ViewResp{
    1:common.resp resp,
}

service webrtc_live {
    BroadcastResp Broadcast(1: BroadcastReq req),
    ViewResp      View(1: ViewReq req),
}
