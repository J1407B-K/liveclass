// internal/rpc/webrtc_live/service/webrtc_live_impl.go
package main

import (
	"context"
	"errors"
	"github.com/cloudwego/kitex/client"
	"github.com/go-redis/redis/v8"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/rpc/webrtc_live/dao"
	"liveclass/internal/rpc/webrtc_live/global"
	my_webrtc "liveclass/internal/rpc/webrtc_live/webrtc"
	"liveclass/internal/utils/cut"
	"log"
	"strconv"
	"time"
)

// WebrtcLiveImpl implements the Kitex webrtc_live service
type WebrtcLiveImpl struct {
	DB        *gorm.DB
	RDB       *redis.Client
	countsha  string
	membersha string
	delsha    string
	selectsha string

	userCli userservice.Client
}

func NewUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r))
}

func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (*webrtc_live.BroadcastResp, error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, _ := cut.SplitInfo(info.Resp.Data)

	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}
	if linfo.Teacher != username {
		return nil, errors.New("你不是该课程老师！无法开播")
	}

	//先解码
	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	//全局配置创建pc
	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	//需要的类型
	pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt, _ := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	vt.SetCodecPreferences([]webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}},
	})

	// 日志
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][Broadcast] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Println("[Server][Broadcast] ICE state:", s.String())
	})

	//收到流时的逻辑
	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println("[Server][Broadcast] OnTrack kind=", remote.Kind())
		local, err := webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, //编码器
			remote.Kind().String()+"-"+req.LessonId, //trackID
			"stream-"+req.LessonId,                  //媒体流名称，归类用
		)
		if err != nil {
			log.Println("track create error:", err)
			return
		}

		//存储
		raw, _ := global.WebRTCEngine.BroadcastTracks.LoadOrStore(req.LessonId, []*webrtc.TrackLocalStaticRTP{})
		arr := raw.([]*webrtc.TrackLocalStaticRTP)
		arr = append(arr, local)
		global.WebRTCEngine.BroadcastTracks.Store(req.LessonId, arr)

		//不断请求关键帧
		go func(ssrc uint32) {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := pc.WriteRTCP([]rtcp.Packet{
					&rtcp.PictureLossIndication{MediaSSRC: ssrc},
				}); err != nil {
					// PC 关闭或出错就停止
					return
				}
			}
		}(uint32(remote.SSRC()))

		for {
			pkt, _, readErr := remote.ReadRTP()
			if readErr != nil {
				return
			}

			if writeErr := local.WriteRTP(pkt); writeErr != nil {
				return
			}
		}
	})

	if err := pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	//连接断开或出错时，从缓存中删除
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][Broadcast] PeerConnection state:", state.String())
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed ||
			state == webrtc.PeerConnectionStateDisconnected {
			log.Println("[Server][Broadcast] Cleaning up maps for lesson", req.LessonId)
			global.WebRTCEngine.BroadcastTracks.Delete(req.LessonId)
		}
	})

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}
	<-webrtc.GatheringCompletePromise(pc)

	//编码
	b64ans, _ := my_webrtc.EncodeSDP(pc.LocalDescription())
	return &webrtc_live.BroadcastResp{Resp: &common.Resp{Data: b64ans}}, nil
}

func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (*webrtc_live.ViewResp, error) {
	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}
	linfo, err := dao.SelectLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}

	for _, stuid := range linfo.StudentID {
		if stuid == req.Userid {
			offer, err := my_webrtc.DecodeSDP(req.B64offer)
			if err != nil {
				return nil, err
			}

			pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
			if err != nil {
				return nil, err
			}

			pc.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c != nil {
					log.Println("[Server][View] ICE candidate:", c.ToJSON())
				}
			})
			pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
				log.Println("[Server][View] ICE state:", s.String())
			})

			if err := pc.SetRemoteDescription(offer); err != nil {
				return nil, err
			}

			raw, ok := global.WebRTCEngine.BroadcastTracks.Load(req.LessonId)
			if !ok {
				return nil, errors.New("未开播: " + req.LessonId)
			}
			//将轨道加入view pc
			for _, t := range raw.([]*webrtc.TrackLocalStaticRTP) {
				pc.AddTrack(t)
			}

			answer, err := pc.CreateAnswer(nil)
			if err != nil {
				return nil, err
			}
			if err := pc.SetLocalDescription(answer); err != nil {
				return nil, err
			}
			<-webrtc.GatheringCompletePromise(pc)

			b64ans, _ := my_webrtc.EncodeSDP(pc.LocalDescription())
			return &webrtc_live.ViewResp{Resp: &common.Resp{Data: b64ans}}, nil
		}
	}
	return nil, errors.New("你不是当前课程学生！")
}

// ChangeUserInLive implements the WebrtcLiveImpl interface.
// 同livego，给前端用的，进入退出直播间直接调
func (s *WebrtcLiveImpl) ChangeUserInLive(ctx context.Context, req *webrtc_live.ChangeUserInLiveReq) (resp *webrtc_live.ChangeUserInLiveResp, err error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)

	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}
	linfo, err := dao.SelectLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}

	var c int
	if req.Options == "add" {
		c = 1
	} else {
		c = -1
	}

	_, err = s.RDB.EvalSha(ctx, s.countsha, []string{req.Lessonid + ":" + linfo.Teacher + ":count"}, c).Result()
	if err != nil {
		return nil, err
	}
	_, err = s.RDB.EvalSha(ctx, s.membersha, []string{req.Lessonid + ":" + linfo.Teacher + ":member"}, req.Options, username, auth).Result()
	if err != nil {
		return nil, err
	}

	return &webrtc_live.ChangeUserInLiveResp{Resp: &common.Resp{
		Data: "success",
	}}, nil
}

// ChangeUserToLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) ChangeUserToLesson(ctx context.Context, req *webrtc_live.ChangeUserToLessonReq) (resp *webrtc_live.ChangeUserToLessonResp, err error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	}

	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}

	if linfo.Teacher != username {
		return nil, errors.New("权限不够！！！你不是老师")
	}

	if req.Options != "add" && req.Options != "del" {
		return nil, errors.New("invalid options")
	}

	err = dao.ChangeUserToLesson(s.DB, lid, req.Userid, req.Options)
	if err != nil {
		return nil, err
	}

	return &webrtc_live.ChangeUserToLessonResp{Resp: &common.Resp{Data: "success"}}, nil
}

// GetLessonInfoById implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetLessonInfoById(ctx context.Context, req *webrtc_live.GetLessonInfoByIdReq) (resp *webrtc_live.GetLessonInfoByIdResp, err error) {
	id, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}
	lesson, err := dao.SelectLesson(s.DB, id)
	if err != nil {
		return nil, err
	}

	var stuidStr string
	for _, uid := range lesson.StudentID {
		stuidStr += uid + "/"
	}

	info := strconv.Itoa(lesson.LessonId) + "$" + lesson.Name + "$" + lesson.Teacher + "$" + lesson.Description + "$" + "/" + stuidStr

	return &webrtc_live.GetLessonInfoByIdResp{
		Resp: &common.Resp{
			Data: info,
		},
	}, nil
}

// CreateLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) CreateLesson(ctx context.Context, req *webrtc_live.CreateLessonReq) (resp *webrtc_live.CreateLessonResp, err error) {
	//请求userrpc拿到userinfo
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！非老师不能创建课程/直播")
	}

	err = dao.CreateLesson(s.DB, req.LessonName, req.Description, username, req.Userid)
	if err != nil {
		return nil, err
	}
	return &webrtc_live.CreateLessonResp{Resp: &common.Resp{
		Data: "success",
	}}, nil
}

// DelLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) DelLesson(ctx context.Context, req *webrtc_live.DelLessonReq) (resp *webrtc_live.DelLessonResp, err error) {
	//请求userrpc拿到userinfo
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！你不是当前课程老师")
	}

	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}
	if linfo.Teacher != username {
		return nil, errors.New("权限不够！你不是当前课程老师")
	}

	_, err = s.RDB.EvalSha(ctx, s.delsha, []string{strconv.Itoa(linfo.LessonId) + ":" + username + ":count", strconv.Itoa(linfo.LessonId) + ":" + username + ":member"}).Result()
	if err != nil {
		return nil, err
	}

	err = dao.DelLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}

	return &webrtc_live.DelLessonResp{Resp: &common.Resp{Data: "success"}}, nil
}
