package tool

import (
	"context"
	"errors"
	"fmt"

	"liveclass/idl/kitex_gen/common"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/toolruntime"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type LessonInfoRequest struct {
	LessonName  string `json:"lesson_name" jsonschema_description:"课程名称"`
	TeacherName string `json:"teacher_name" jsonschema_description:"授课老师名称"`
}

type LessonInfoResponse struct {
	LessonID    int64   `json:"lesson_id"`
	Name        string  `json:"name"`
	TeacherName string  `json:"teacher_name"`
	TeacherID   int64   `json:"teacher_id"`
	Description string  `json:"description"`
	StudentIDs  []int64 `json:"student_ids"`
	StudentCnt  int     `json:"student_count"`
}

func NewLessonInfoTool(cli webrtclive.Client) (tool.InvokableTool, error) {
	if cli == nil {
		return nil, errors.New("nil lesson client")
	}

	call := func(ctx context.Context, req *LessonInfoRequest) (*LessonInfoResponse, error) {
		if req == nil {
			return nil, errors.New("empty request")
		}
		if req.LessonName == "" || req.TeacherName == "" {
			return nil, errors.New("lesson_name 与 teacher_name 均不能为空")
		}

		lesson, err := fetchLessonByName(ctx, cli, req.LessonName, req.TeacherName)
		if err != nil {
			return nil, err
		}
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok || principal.LessonID != lesson.GetLessonID() || (principal.UserID != lesson.GetTeacherID() && !containsUserID(lesson.GetStudentID(), principal.UserID)) {
			return nil, errors.New("lesson permission denied")
		}

		return &LessonInfoResponse{
			LessonID:    lesson.GetLessonID(),
			Name:        lesson.GetName(),
			TeacherName: lesson.GetTeacherName(),
			TeacherID:   lesson.GetTeacherID(),
			Description: lesson.GetDescription(),
			StudentIDs:  lesson.GetStudentID(),
			StudentCnt:  len(lesson.GetStudentID()),
		}, nil
	}

	return utils.InferTool("lesson_info_lookup", "根据课程名称与教师姓名查询课程信息", call)
}

func containsUserID(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func fetchLessonByName(ctx context.Context, cli webrtclive.Client, lessonName, teacherName string) (*common.Lesson, error) {
	resp, err := dependency.Do(ctx, dependency.InternalRPC, "get_lesson_info", func(callCtx context.Context) (*webrtc_live.GetLessonInfoResp, error) {
		return cli.GetLessonInfo(callCtx, &webrtc_live.GetLessonInfoReq{
			LessonName: lessonName,
			Teacher:    teacherName,
		})
	})
	if err != nil {
		return nil, err
	}
	return parseLessonResp(resp)
}

func parseLessonResp(resp *webrtc_live.GetLessonInfoResp) (*common.Lesson, error) {
	if resp == nil || resp.GetResp() == nil {
		return nil, errors.New("lesson service: empty response")
	}
	if resp.GetResp().GetCode() != 0 {
		return nil, fmt.Errorf("lesson service error: %s", resp.GetResp().GetMsg())
	}
	data := resp.GetResp().GetData()
	if data == nil || data.GetLessonInfo() == nil {
		return nil, errors.New("lesson service: missing lesson info")
	}
	return data.GetLessonInfo(), nil
}
