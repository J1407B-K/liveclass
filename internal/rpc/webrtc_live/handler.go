// internal/rpc/webrtc_live/service/webrtc_live_impl.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	my_cos "liveclass/internal/rpc/webrtc_live/cos"
	"liveclass/internal/rpc/webrtc_live/dao"
	"liveclass/internal/rpc/webrtc_live/global"
	"liveclass/internal/rpc/webrtc_live/model"
	my_webrtc "liveclass/internal/rpc/webrtc_live/webrtc"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"github.com/tencentyun/cos-go-sdk-v5"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// WebrtcLiveImpl implements the Kitex webrtc_live service
type WebrtcLiveImpl struct {
	DBManager *dao.DBManager
	cosClient *cos.Client
	changesha string
	delsha    string
	selectsha string

	userCli userservice.Client

	sfLesson singleflight.Group
}

var ErrLessonNotExist = errors.New("lesson not exist")

func NewUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(800*time.Millisecond))
}

func (s *WebrtcLiveImpl) Broadcast(ctx context.Context, req *webrtc_live.BroadcastReq) (*webrtc_live.BroadcastResp, error) {
	_, _, err := s.requireTeacherOfLesson(ctx, req.LessonId, req.Userid)
	if err != nil {
		return nil, err
	}

	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	ok := false

	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !ok {
			_ = pc.Close()
		}
	}()

	if _, err = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		return nil, err
	}

	vt, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	)
	if err != nil {
		return nil, err
	}

	if err = vt.SetCodecPreferences([]webrtc.RTPCodecParameters{
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}},
		{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000}},
	}); err != nil {
		return nil, err
	}

	sessionID := uuid.NewString()

	// 如果已有旧直播，先尝试关闭旧 publisher
	if oldRaw, ok := global.WebRTCEngine.BroadcastRooms.Load(req.LessonId); ok {
		if oldBundle, ok2 := oldRaw.(*model.BroadcastBundle); ok2 && oldBundle.PublisherPC != nil {
			_ = oldBundle.PublisherPC.Close()
		}
	}

	bundle := &model.BroadcastBundle{
		SessionID:       sessionID,
		PublisherPC:     pc,
		PublisherStatus: model.ConnectionConnecting,
	}
	global.WebRTCEngine.BroadcastRooms.Store(req.LessonId, bundle)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][Broadcast] ICE candidate:", c.ToJSON())
		}
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Println("[Server][Broadcast] ICE state:", state.String())
	})

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println("[Server][Broadcast] OnTrack kind=", remote.Kind())

		trackID := fmt.Sprintf("%s-%d-%d", remote.Kind().String(), req.LessonId, remote.SSRC())
		streamID := fmt.Sprintf("stream-%d", req.LessonId)

		local, err := webrtc.NewTrackLocalStaticRTP(
			remote.Codec().RTPCodecCapability,
			trackID,
			streamID,
		)
		if err != nil {
			log.Println("[Server][Broadcast] track create error:", err)
			return
		}

		raw, ok := global.WebRTCEngine.BroadcastRooms.Load(req.LessonId)
		if !ok {
			return
		}
		b, ok := raw.(*model.BroadcastBundle)
		if !ok {
			return
		}
		if b.SessionID != sessionID {
			return
		}

		b.Mu.Lock()
		switch remote.Kind() {
		case webrtc.RTPCodecTypeVideo:
			b.VideoTrack = local
		case webrtc.RTPCodecTypeAudio:
			b.AudioTrack = local
		}
		b.Mu.Unlock()

		if remote.Kind() == webrtc.RTPCodecTypeVideo {
			go func(ssrc uint32) {
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := pc.WriteRTCP([]rtcp.Packet{
							&rtcp.PictureLossIndication{MediaSSRC: ssrc},
						}); err != nil {
							return
						}
					}
				}
			}(uint32(remote.SSRC()))
		}

		go func() {
			for {
				pkt, _, readErr := remote.ReadRTP()
				if readErr != nil {
					log.Println("[Server][Broadcast] remote.ReadRTP error:", readErr)
					return
				}
				if writeErr := local.WriteRTP(pkt); writeErr != nil {
					log.Println("[Server][Broadcast] local.WriteRTP error:", writeErr)
					return
				}
			}
		}()
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][Broadcast] PeerConnection state:", state.String())

		raw, ok := global.WebRTCEngine.BroadcastRooms.Load(req.LessonId)
		if ok {
			if b, ok2 := raw.(*model.BroadcastBundle); ok2 && b.SessionID == sessionID {
				b.Mu.Lock()
				switch state {
				case webrtc.PeerConnectionStateConnecting:
					b.PublisherStatus = model.ConnectionConnecting
				case webrtc.PeerConnectionStateConnected:
					b.PublisherStatus = model.ConnectionConnected
				case webrtc.PeerConnectionStateDisconnected:
					b.PublisherStatus = model.ConnectionDisconnected
				case webrtc.PeerConnectionStateFailed:
					b.PublisherStatus = model.ConnectionFailed
				case webrtc.PeerConnectionStateClosed:
					b.PublisherStatus = model.ConnectionClosed
				}
				b.Mu.Unlock()
			}
		}

		// 注意：disconnected 不立刻删
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			if raw, ok := global.WebRTCEngine.BroadcastRooms.Load(req.LessonId); ok {
				if b, ok2 := raw.(*model.BroadcastBundle); ok2 && b.SessionID == sessionID {
					log.Println("[Server][Broadcast] Cleaning up maps for lesson", req.LessonId, "session", sessionID)
					global.WebRTCEngine.BroadcastRooms.Delete(req.LessonId)
				}
			}
			_ = pc.Close()
		}
	})

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}

	ok = true
	return &webrtc_live.BroadcastResp{
		Resp: &common.Resp{
			Data: &common.Data{Sdp: strptr(b64ans)},
		},
	}, nil
}

func (s *WebrtcLiveImpl) View(ctx context.Context, req *webrtc_live.ViewReq) (*webrtc_live.ViewResp, error) {
	_, err := s.ensureLessonExists(ctx, req.LessonId)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.ViewResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.LessonId, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("你不是当前课程学生！")
	}

	log.Println("[Server][View] req lessonId =", req.LessonId, "uid =", req.Userid)

	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}
	ok = false
	defer func() {
		if !ok {
			_ = pc.Close()
		}
	}()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][View] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Println("[Server][View] ICE state:", state.String())
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][View] PeerConnection state:", state.String())
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			_ = pc.Close()
		}
	})

	raw, ok2 := global.WebRTCEngine.BroadcastRooms.Load(req.LessonId)
	if !ok2 {
		return nil, errors.New("未开播: " + strconv.FormatInt(req.LessonId, 10))
	}
	b, ok2 := raw.(*model.BroadcastBundle)
	if !ok2 {
		return nil, errors.New("broadcast bundle type error")
	}

	var videoTrack *webrtc.TrackLocalStaticRTP
	var audioTrack *webrtc.TrackLocalStaticRTP

	deadline := time.NewTimer(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()

	for {
		b.Mu.RLock()
		status := b.PublisherStatus
		videoTrack = b.VideoTrack
		audioTrack = b.AudioTrack
		b.Mu.RUnlock()

		if status == model.ConnectionFailed || status == model.ConnectionClosed {
			return nil, errors.New("直播已结束")
		}

		// 有视频就可以开看；音频可选
		if videoTrack != nil {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errors.New("老师视频轨尚未就绪，请稍后再试")
		case <-ticker.C:
		}
	}

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	log.Println("[Server][View] AddTrack video:", videoTrack.ID(), videoTrack.StreamID())
	videoSender, err := pc.AddTrack(videoTrack)
	if err != nil {
		return nil, err
	}
	go func(s *webrtc.RTPSender) {
		buf := make([]byte, 1500)
		for {
			n, _, readErr := s.Read(buf)
			if readErr != nil {
				log.Println("[Server][View] video RTCP Read error:", readErr)
				return
			}
			_, _ = rtcp.Unmarshal(buf[:n])
		}
	}(videoSender)

	if audioTrack != nil {
		log.Println("[Server][View] AddTrack audio:", audioTrack.ID(), audioTrack.StreamID())
		audioSender, err := pc.AddTrack(audioTrack)
		if err != nil {
			return nil, err
		}
		go func(s *webrtc.RTPSender) {
			buf := make([]byte, 1500)
			for {
				n, _, readErr := s.Read(buf)
				if readErr != nil {
					log.Println("[Server][View] audio RTCP Read error:", readErr)
					return
				}
				_, _ = rtcp.Unmarshal(buf[:n])
			}
		}(audioSender)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}

	ok = true
	return &webrtc_live.ViewResp{
		Resp: &common.Resp{
			Data: &common.Data{Sdp: strptr(b64ans)},
		},
	}, nil
}

// ChangeUserInLive implements the WebrtcLiveImpl interface.
// 同livego，给前端用的，进入退出直播间直接调
func (s *WebrtcLiveImpl) ChangeUserInLive(ctx context.Context, req *webrtc_live.ChangeUserInLiveReq) (resp *webrtc_live.ChangeUserInLiveResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.ChangeUserInLiveResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	userinfo, err := s.getUserInfo(ctx, req.Userid)
	if err != nil {
		return nil, err
	}

	countKey := dao.LiveCountKey(req.Lessonid)
	memberKey := dao.LiveMembersKey(req.Lessonid)

	_, err = s.DBManager.RDB.EvalSha(ctx, s.changesha,
		[]string{countKey, memberKey},
		req.Options, req.Userid, userinfo.Auth,
	).Result()
	if err != nil {
		return nil, err
	}

	return &webrtc_live.ChangeUserInLiveResp{Resp: &common.Resp{
		Msg: "success",
	}}, nil
}

// ChangeUserToLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) ChangeUserToLesson(ctx context.Context, req *webrtc_live.ChangeUserToLessonReq) (*webrtc_live.ChangeUserToLessonResp, error) {
	_, _, err := s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	if req.Options != "add" && req.Options != "del" {
		return nil, errors.New("invalid options")
	}

	if err = s.DBManager.ChangeUserToLesson(req.Lessonid, req.Stuid, req.Options); err != nil {
		return nil, err
	}

	return &webrtc_live.ChangeUserToLessonResp{Resp: &common.Resp{Msg: "success"}}, nil
}

// GetLessonInfoById implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetLessonInfoById(ctx context.Context, req *webrtc_live.GetLessonInfoByIdReq) (*webrtc_live.GetLessonInfoByIdResp, error) {
	lesson, err := s.DBManager.SelectLesson(req.Lessonid)
	if err != nil {
		return nil, err
	}

	ids, err := s.DBManager.ListLessonStudentIDs(lesson.LessonId)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	for i, uid := range ids {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(strconv.FormatInt(uid, 10))
	}

	return &webrtc_live.GetLessonInfoByIdResp{
		Resp: &common.Resp{
			Data: &common.Data{
				LessonInfo: &common.Lesson{
					LessonID:    lesson.LessonId,
					Name:        lesson.Name,
					TeacherName: lesson.TeacherName,
					Description: lesson.Description,
					StudentID:   ids,
					TeacherID:   lesson.TeacherUID,
				},
			},
		},
	}, nil
}

// CreateLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) CreateLesson(ctx context.Context, req *webrtc_live.CreateLessonReq) (resp *webrtc_live.CreateLessonResp, err error) {
	userinfo, err := s.getUserInfo(ctx, req.Userid)
	if err != nil {
		return nil, err
	}

	if userinfo.Auth != "Teacher" {
		return nil, errors.New("权限不够！非老师不能创建课程/直播")
	}

	lessonid, err := s.DBManager.CreateLesson(req.LessonName, req.Description, userinfo.UserName, userinfo.Auth, req.Userid)
	if err != nil {
		return nil, err
	}

	_ = s.DBManager.RDB.Del(ctx, fmt.Sprintf("lesson:info:%d", lessonid)).Err()

	err = s.DBManager.AddBloom(ctx, lessonid)
	if err != nil {
		return nil, err
	}

	return &webrtc_live.CreateLessonResp{Resp: &common.Resp{
		Code: 0,
		Msg:  "success",
		Data: &common.Data{
			LessonInfo: &common.Lesson{
				LessonID: lessonid,
			},
		},
	}}, nil
}

// DelLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) DelLesson(ctx context.Context, req *webrtc_live.DelLessonReq) (resp *webrtc_live.DelLessonResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	if _, err = s.DBManager.RDB.EvalSha(ctx, s.delsha, []string{
		dao.LiveCountKey(req.Lessonid),
		dao.LiveMembersKey(req.Lessonid),
	}).Result(); err != nil {
		return nil, err
	}

	err = s.DBManager.DelLesson(req.Lessonid)
	if err != nil {
		return nil, err
	}

	_ = s.DBManager.RDB.Del(ctx, fmt.Sprintf("lesson:info:%d", req.Lessonid)).Err()

	return &webrtc_live.DelLessonResp{Resp: &common.Resp{Msg: "success"}}, nil
}

// SelectLessonInfo implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) SelectLessonInfo(ctx context.Context, req *webrtc_live.SelectLessonInfoReq) (resp *webrtc_live.SelectLessonInfoResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.SelectLessonInfoResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	countKey := dao.LiveCountKey(req.Lessonid)
	memberKey := dao.LiveMembersKey(req.Lessonid)

	r, err := s.DBManager.RDB.EvalSha(ctx, s.selectsha, []string{countKey, memberKey}).Result()
	if err != nil {
		return nil, err
	}

	ar, ok := r.([]interface{})
	if !ok {
		return nil, errors.New("解析redis lessonInfo 失败")
	}

	countStr := ar[0].(string)

	var membersStr string
	for i := 1; i < len(ar); i++ {
		membersStr += ar[i].(string)
		if i%2 == 0 {
			membersStr += "  "
		} else {
			membersStr += "$"
		}
	}

	return &webrtc_live.SelectLessonInfoResp{
		Resp: &common.Resp{Msg: "count:" + countStr + "///" + "live member:" + membersStr},
	}, nil
}

// GetLessonInfo implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetLessonInfo(ctx context.Context, req *webrtc_live.GetLessonInfoReq) (resp *webrtc_live.GetLessonInfoResp, err error) {
	linfo, err := s.DBManager.SelectLessonByNandT(req.LessonName, req.Teacher)
	if err != nil {
		return nil, err
	}
	stuid, err := s.DBManager.ListLessonStudentIDs(linfo.LessonId)
	if err != nil {
		return nil, err
	}

	return &webrtc_live.GetLessonInfoResp{Resp: &common.Resp{
		Data: &common.Data{
			LessonInfo: &common.Lesson{
				LessonID:    linfo.LessonId,
				Name:        linfo.Name,
				TeacherName: linfo.TeacherName,
				Description: linfo.Description,
				StudentID:   stuid,
			},
		},
	}}, nil
}

// IsStudentInLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) IsStudentInLesson(ctx context.Context, req *webrtc_live.IsStudentInLessonReq) (resp *webrtc_live.IsStudentInLessonResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.IsStudentInLessonResp{Resp: &common.Resp{Msg: "not_exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Studentid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &webrtc_live.IsStudentInLessonResp{
			Resp: &common.Resp{
				Msg: "not_exist",
			},
		}, nil
	}
	return &webrtc_live.IsStudentInLessonResp{
		Resp: &common.Resp{
			Msg: "exist",
		},
	}, nil
}

// CreateSignIn implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) CreateSignIn(ctx context.Context, req *webrtc_live.CreateSignInReq) (resp *webrtc_live.CreateSignInResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.CreateSignInResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	linfo, err := s.DBManager.SelectLesson(req.Lessonid)
	if err != nil {
		return nil, err
	}

	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	stuid, err := s.DBManager.ListLessonStudentIDs(linfo.LessonId)
	if err != nil {
		return nil, err
	}

	err = s.DBManager.CreateSignIn(req.Lessonid, stuid, time.Now().Add(time.Duration(req.Duration)*time.Second))
	if err != nil {
		return nil, err
	}
	return &webrtc_live.CreateSignInResp{
		Resp: &common.Resp{
			Msg: "success",
		},
	}, nil
}

// SignIn implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) SignIn(ctx context.Context, req *webrtc_live.SignInReq) (*webrtc_live.SignInResp, error) {
	_, err := s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.SignInResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("不是此课程学生")
	}

	_, err = s.DBManager.StuSignIn(req.Lessonid, req.Userid, time.Now())
	if err != nil {
		if err.Error() == "close" {
			return &webrtc_live.SignInResp{Resp: &common.Resp{Msg: "签到关闭"}}, nil
		}
		return nil, err
	}

	return &webrtc_live.SignInResp{Resp: &common.Resp{Msg: "success"}}, nil
}

// SelectSignIn implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) SelectSignIn(ctx context.Context, req *webrtc_live.SelectSignInReq) (resp *webrtc_live.SelectSignInResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	sinfo, err := s.DBManager.SelectSignIn(req.Lessonid)
	if err != nil {
		return nil, err
	}

	return &webrtc_live.SelectSignInResp{Resp: &common.Resp{Msg: sinfo}}, nil
}

// DelSign implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) DelSign(ctx context.Context, req *webrtc_live.DelSignInReq) (resp *webrtc_live.DelSignInResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	err = s.DBManager.RemoveSignIn(req.Lessonid)
	if err != nil {
		return nil, err
	}
	return &webrtc_live.DelSignInResp{Resp: &common.Resp{Msg: "success"}}, nil
}

// RollCallInRandom implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) RollCallInRandom(ctx context.Context, req *webrtc_live.RollCallInRandomReq) (resp *webrtc_live.RollCallInRandomResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.LessonId, req.Userid)
	if err != nil {
		return nil, err
	}

	stuid, err := s.DBManager.ListLessonStudentIDs(req.LessonId)
	if err != nil {
		return nil, err
	}

	randomIndex := rand.Intn(len(stuid))

	stuinfo, err := s.getUserInfo(ctx, stuid[randomIndex])
	if err != nil {
		return nil, err
	}

	return &webrtc_live.RollCallInRandomResp{Resp: &common.Resp{Msg: stuinfo.UserName}}, nil
}

// RecordLesson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) RecordLesson(ctx context.Context, req *webrtc_live.RecordLessonReq) (resp *webrtc_live.RecordLessonResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	err = os.Mkdir(global.Config.TmpBaseDir, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	filename := fmt.Sprintf("%d-record-%s.mp4", req.Lessonid, uuid.NewString())
	localfile := filepath.Join(global.Config.TmpBaseDir, filename)

	if len(req.Data) == 0 {
		return nil, errors.New("没有任何数据需要写入")
	}
	if err := os.WriteFile(localfile, req.Data, 0o644); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}

	go func() {
		if err := my_cos.UploadToCos(ctx, s.cosClient, localfile, strconv.FormatInt(req.Lessonid, 10), filename); err != nil {
			log.Printf("上传到 COS 失败: %v", err)
		} else {
			log.Printf("上传到 COS 成功: lesson=%d file=%s", req.Lessonid, filename)
		}
		if rmErr := os.Remove(localfile); rmErr != nil {
			log.Printf("删除临时文件失败: %v", rmErr)
		}
	}()

	return &webrtc_live.RecordLessonResp{
		Resp: &common.Resp{Msg: filename},
	}, nil
}

// SaveWhiteBoardJson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) SaveWhiteBoardJson(ctx context.Context, req *webrtc_live.SaveWhiteBoardJsonReq) (resp *webrtc_live.SaveWhiteBoardJsonResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	err = s.DBManager.SaveWhiteBoard(req.Lessonid, req.File)
	if err != nil {
		return nil, err
	}
	return &webrtc_live.SaveWhiteBoardJsonResp{Resp: &common.Resp{Msg: "success"}}, nil
}

// GetWhiteBoardJson implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetWhiteBoardJson(ctx context.Context, req *webrtc_live.GetWhiteBoardJsonReq) (resp *webrtc_live.GetWhiteBoardJsonResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	docs, err := s.DBManager.GetWhiteBoardNew(req.Lessonid)
	if docs == nil && err == nil {
		return &webrtc_live.GetWhiteBoardJsonResp{Resp: &common.Resp{Msg: "not_found"}}, nil
	}
	if err != nil {
		return nil, err
	}

	docByte, err := json.Marshal(docs)
	if err != nil {
		return nil, err
	}
	return &webrtc_live.GetWhiteBoardJsonResp{Resp: &common.Resp{Data: &common.Data{Text: strptr(string(docByte))}}}, nil
}

// PublishMic implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) PublishMic(ctx context.Context, req *webrtc_live.PublishMicReq) (resp *webrtc_live.PublishMicResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.PublishMicResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("你不是本课程学生")
	}

	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	ok = false
	defer func() {
		if !ok {
			_ = pc.Close()
		}
	}()

	_, err = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		return nil, err
	}

	sessionID := "mic-" + uuid.NewString()
	bundle := s.getOrCreateMicBundle(req.Lessonid)

	// 同一个学生重复上麦时，先关掉旧连接
	bundle.Mu.Lock()
	if oldPub, exists := bundle.Publishers[req.Userid]; exists && oldPub.PC != nil {
		_ = oldPub.PC.Close()
	}
	bundle.Publishers[req.Userid] = &model.MicPublisher{
		UserID:    req.Userid,
		SessionID: sessionID,
		PC:        pc,
		Status:    model.ConnectionConnecting,
	}
	bundle.Mu.Unlock()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][PublishMic] ICE candidate:", c.ToJSON())
		}
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Println("[Server][PublishMic] ICE state:", state.String())
	})

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Println("[Server][PublishMic] OnTrack kind=", remote.Kind())

		if remote.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}

		local, err := webrtc.NewTrackLocalStaticRTP(
			remote.Codec().RTPCodecCapability,
			"mic-"+strconv.FormatInt(req.Lessonid, 10)+"-"+strconv.FormatInt(req.Userid, 10),
			"stream-"+strconv.FormatInt(req.Lessonid, 10),
		)
		if err != nil {
			log.Println("[Server][PublishMic] track create error:", err)
			return
		}

		raw, ok := global.WebRTCEngine.MicTracks.Load(req.Lessonid)
		if !ok {
			return
		}
		b, ok := raw.(*model.MicBundle)
		if !ok {
			return
		}

		b.Mu.Lock()
		pub, exists := b.Publishers[req.Userid]
		if !exists || pub.SessionID != sessionID {
			b.Mu.Unlock()
			return
		}
		pub.Track = local
		b.Mu.Unlock()

		go func() {
			for {
				pkt, _, readErr := remote.ReadRTP()
				if readErr != nil {
					log.Println("[Server][PublishMic] remote.ReadRTP error:", readErr)
					return
				}
				if writeErr := local.WriteRTP(pkt); writeErr != nil {
					log.Println("[Server][PublishMic] local.WriteRTP error:", writeErr)
					return
				}
			}
		}()
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][PublishMic] PeerConnection state:", state.String())

		raw, ok := global.WebRTCEngine.MicTracks.Load(req.Lessonid)
		if ok {
			if b, ok2 := raw.(*model.MicBundle); ok2 {
				b.Mu.Lock()
				if pub, exists := b.Publishers[req.Userid]; exists && pub.SessionID == sessionID {
					switch state {
					case webrtc.PeerConnectionStateConnecting:
						pub.Status = model.ConnectionConnecting
					case webrtc.PeerConnectionStateConnected:
						pub.Status = model.ConnectionConnected
					case webrtc.PeerConnectionStateDisconnected:
						pub.Status = model.ConnectionDisconnected
					case webrtc.PeerConnectionStateFailed:
						pub.Status = model.ConnectionFailed
					case webrtc.PeerConnectionStateClosed:
						pub.Status = model.ConnectionClosed
					}
				}
				b.Mu.Unlock()
			}
		}

		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			s.removeMicPublisher(req.Lessonid, req.Userid, sessionID)
			_ = pc.Close()
		}
	})

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	ans, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err = pc.SetLocalDescription(ans); err != nil {
		return nil, err
	}

	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}

	ok = true
	return &webrtc_live.PublishMicResp{
		Resp: &common.Resp{
			Data: &common.Data{Sdp: strptr(b64ans)},
		},
	}, nil
}

// RaiseHand implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) RaiseHand(ctx context.Context, req *webrtc_live.RaiseHandReq) (resp *webrtc_live.RaiseHandResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.RaiseHandResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &webrtc_live.RaiseHandResp{
			Resp: &common.Resp{Msg: "不是本课程学生"},
		}, nil
	}

	key := dao.HandsKey(req.Lessonid)
	member := strconv.FormatInt(req.Userid, 10)
	score := float64(time.Now().UnixMilli())

	if err = s.DBManager.RDB.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return nil, err
	}
	_ = s.DBManager.RDB.Expire(ctx, key, 24*time.Hour).Err()

	return &webrtc_live.RaiseHandResp{
		Resp: &common.Resp{Msg: "success"},
	}, nil
}

// GetRaiseHand implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetRaiseHand(ctx context.Context, req *webrtc_live.GetRaiseHandReq) (resp *webrtc_live.GetRaiseHandResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	key := dao.HandsKey(req.Lessonid)

	ids, err := s.DBManager.RDB.ZRange(ctx, key, 0, 99).Result()
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return &webrtc_live.GetRaiseHandResp{
			Resp: &common.Resp{Msg: "当前无举手学生"},
		}, nil
	}

	return &webrtc_live.GetRaiseHandResp{
		Resp: &common.Resp{Msg: strings.Join(ids, "/")},
	}, nil
}

// ApproveHand implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) ApproveHand(ctx context.Context, req *webrtc_live.ApproveHandReq) (resp *webrtc_live.ApproveHandResp, err error) {
	_, _, err = s.requireTeacherOfLesson(ctx, req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}

	key := dao.HandsKey(req.Lessonid)
	member := strconv.FormatInt(req.Stuid, 10)

	removed, err := s.DBManager.RDB.ZRem(ctx, key, member).Result()
	if err != nil {
		return nil, err
	}
	if removed == 0 {
		return &webrtc_live.ApproveHandResp{
			Resp: &common.Resp{Msg: "该学生没有举手"},
		}, nil
	}

	return &webrtc_live.ApproveHandResp{
		Resp: &common.Resp{Msg: "success"},
	}, nil
}

// ViewMic implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) ViewMic(ctx context.Context, req *webrtc_live.ViewMicReq) (resp *webrtc_live.ViewMicResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.ViewMicResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("你不是本课程学生")
	}

	offer, err := my_webrtc.DecodeSDP(req.B64offer)
	if err != nil {
		return nil, err
	}

	pc, err := global.WebRTCEngine.API.NewPeerConnection(global.WebRTCEngine.SfuConfig)
	if err != nil {
		return nil, err
	}

	ok = false
	defer func() {
		if !ok {
			_ = pc.Close()
		}
	}()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			log.Println("[Server][ViewMic] ICE candidate:", c.ToJSON())
		}
	})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Println("[Server][ViewMic] ICE state:", state.String())
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("[Server][ViewMic] PeerConnection state:", state.String())
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			_ = pc.Close()
		}
	})

	raw, ok := global.WebRTCEngine.MicTracks.Load(req.Lessonid)
	if !ok {
		return &webrtc_live.ViewMicResp{
			Resp: &common.Resp{Msg: "目前无任何学生上麦"},
		}, nil
	}

	b, ok := raw.(*model.MicBundle)
	if !ok {
		return nil, errors.New("mic bundle type error")
	}

	var tracks []*webrtc.TrackLocalStaticRTP

	b.Mu.RLock()
	for _, pub := range b.Publishers {
		if pub == nil || pub.Track == nil {
			continue
		}
		if pub.Status == model.ConnectionFailed || pub.Status == model.ConnectionClosed {
			continue
		}
		tracks = append(tracks, pub.Track)
	}
	b.Mu.RUnlock()

	if len(tracks) == 0 {
		return &webrtc_live.ViewMicResp{
			Resp: &common.Resp{Msg: "目前无任何学生上麦"},
		}, nil
	}

	if err = pc.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	for _, t := range tracks {
		log.Println("[Server][ViewMic] AddTrack:", t.ID(), t.StreamID())

		sender, err := pc.AddTrack(t)
		if err != nil {
			return nil, err
		}

		go func(s *webrtc.RTPSender) {
			buf := make([]byte, 1500)
			for {
				n, _, readErr := s.Read(buf)
				if readErr != nil {
					log.Println("[Server][ViewMic] RTCP Read error:", readErr)
					return
				}
				_, _ = rtcp.Unmarshal(buf[:n])
			}
		}(sender)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-webrtc.GatheringCompletePromise(pc)

	b64ans, err := my_webrtc.EncodeSDP(pc.LocalDescription())
	if err != nil {
		return nil, err
	}

	ok = true
	return &webrtc_live.ViewMicResp{
		Resp: &common.Resp{
			Data: &common.Data{Sdp: strptr(b64ans)},
		},
	}, nil
}

// ListAllLessonRecord implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) ListAllLessonRecord(ctx context.Context, req *webrtc_live.ListAllLessonRecordReq) (resp *webrtc_live.ListAllLessonRecordResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.ListAllLessonRecordResp{Resp: &common.Resp{Msg: "not exist"}}, nil
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("你不是本课程学生")
	}
	prifix := "lesson_" + strconv.FormatInt(req.Lessonid, 10)

	opt := &cos.BucketGetOptions{
		MaxKeys: 100,
		Prefix:  prifix,
	}
	result, _, err := s.cosClient.Bucket.Get(ctx, opt)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	for i, v := range result.Contents {
		sizeKB := float64(v.Size) / 1024
		b.WriteString(fmt.Sprintf("[%d] %s (%.1f KB)\n", i, v.Key, sizeKB))
	}
	return &webrtc_live.ListAllLessonRecordResp{Resp: &common.Resp{Msg: b.String()}}, nil
}

// GetLessonRecord implements the WebrtcLiveImpl interface.
func (s *WebrtcLiveImpl) GetLessonRecord(ctx context.Context, req *webrtc_live.GetLessonRecordReq) (resp *webrtc_live.GetLessonRecordResp, err error) {
	_, err = s.ensureLessonExists(ctx, req.Lessonid)
	if err != nil {
		if errors.Is(err, ErrLessonNotExist) {
			return &webrtc_live.GetLessonRecordResp{Data: nil}, ErrLessonNotExist
		}
		return nil, err
	}

	ok, err := s.DBManager.IsStudentInLesson(req.Lessonid, req.Userid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("你不是本课程学生")
	}
	key := "lesson_" + strconv.FormatInt(req.Lessonid, 10) + "/" + req.Key

	filename := req.Key
	localfile := filepath.Join(global.Config.TmpBaseDir, filename)

	data, err := my_cos.DownloadFromCos(ctx, s.cosClient, localfile, key)
	if err != nil {
		return nil, err
	}
	if rmErr := os.Remove(localfile); rmErr != nil {
		log.Printf("删除临时文件失败: %v", rmErr)
	}

	return &webrtc_live.GetLessonRecordResp{Data: data}, nil
}

func (s *WebrtcLiveImpl) getUserInfo(ctx context.Context, userID int64) (*common.User, error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: userID})
	if err != nil {
		return nil, err
	}
	if info == nil || info.Resp == nil || info.Resp.Data == nil {
		return nil, errors.New("user service: empty userinfo")
	}
	return info.Resp.Data.UserInfo, nil
}

func (s *WebrtcLiveImpl) mustLesson(ctx context.Context, lessonID int64) (*model.WebrtcLesson, error) {
	return s.ensureLessonExists(ctx, lessonID)
}

func (s *WebrtcLiveImpl) requireTeacherOfLesson(ctx context.Context, lessonID int64, userID int64) (*model.WebrtcLesson, *common.User, error) {
	cacheKey := fmt.Sprintf("perm:teacher:%d:%d", lessonID, userID)

	if s.DBManager.RDB != nil {
		if val, _ := s.DBManager.RDB.Get(ctx, cacheKey).Result(); val == "1" {
			u, err := s.getUserInfo(ctx, userID)
			if err != nil {
				return nil, nil, err
			}
			lesson, err := s.mustLesson(ctx, lessonID)
			if err != nil {
				return nil, nil, err
			}
			return lesson, u, nil
		}
	}

	u, err := s.getUserInfo(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	lesson, err := s.mustLesson(ctx, lessonID)
	if err != nil {
		return nil, nil, err
	}

	if lesson.TeacherUID != u.UserID {
		return nil, nil, errors.New("权限不够：你不是该课程老师")
	}

	if s.DBManager.RDB != nil {
		_ = s.DBManager.RDB.Set(ctx, cacheKey, "1", 5*time.Minute).Err()
	}
	return lesson, u, nil
}

func (s *WebrtcLiveImpl) ensureLessonExists(ctx context.Context, lessonID int64) (*model.WebrtcLesson, error) {
	maybe, err := s.DBManager.BloomMaybeLesson(ctx, lessonID)
	if err == nil && !maybe {
		return nil, ErrLessonNotExist
	}

	lesson, err := s.getLessonCached(ctx, lessonID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLessonNotExist
		}
		return nil, err
	}
	return lesson, nil
}

func (s *WebrtcLiveImpl) getLessonCached(ctx context.Context, lessonID int64) (*model.WebrtcLesson, error) {
	cacheKey := fmt.Sprintf("lesson:info:%d", lessonID)

	if s.DBManager.RDB != nil {
		b, err := s.DBManager.RDB.Get(ctx, cacheKey).Bytes()
		if err == nil && len(b) > 0 {
			var l model.WebrtcLesson
			if jsonErr := json.Unmarshal(b, &l); jsonErr == nil {
				return &l, nil
			}
			_ = s.DBManager.RDB.Del(ctx, cacheKey).Err()
		}
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}

	v, err, _ := s.sfLesson.Do(cacheKey, func() (any, error) {
		lesson, err := s.DBManager.SelectLesson(lessonID)
		if err != nil {
			return nil, err
		}

		if s.DBManager.RDB != nil {
			if b, jerr := json.Marshal(lesson); jerr == nil {
				ttl := 5*time.Minute + time.Duration(rand.Intn(30))*time.Second
				_ = s.DBManager.RDB.Set(ctx, cacheKey, b, ttl).Err()
			}
		}
		return lesson, nil
	})
	if err != nil {
		return nil, err
	}
	lesson, ok := v.(*model.WebrtcLesson)
	if !ok || lesson == nil {
		return nil, errors.New("invalid lesson type")
	}
	return lesson, nil
}

func (s *WebrtcLiveImpl) getOrCreateMicBundle(lessonID int64) *model.MicBundle {
	raw, _ := global.WebRTCEngine.MicTracks.LoadOrStore(lessonID, &model.MicBundle{
		Publishers: make(map[int64]*model.MicPublisher),
	})
	return raw.(*model.MicBundle)
}

func (s *WebrtcLiveImpl) removeMicPublisher(lessonID, userID int64, sessionID string) {
	raw, ok := global.WebRTCEngine.MicTracks.Load(lessonID)
	if !ok {
		return
	}
	b, ok := raw.(*model.MicBundle)
	if !ok {
		return
	}

	b.Mu.Lock()
	defer b.Mu.Unlock()

	pub, ok := b.Publishers[userID]
	if !ok {
		return
	}
	if pub.SessionID != sessionID {
		return
	}

	delete(b.Publishers, userID)

	if len(b.Publishers) == 0 {
		global.WebRTCEngine.MicTracks.Delete(lessonID)
	}
}

func strptr(s string) *string { return &s }
