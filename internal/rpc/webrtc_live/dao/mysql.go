package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/internal/rpc/webrtc_live/model"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (m *DBManager) CreateLesson(name, desc, teacherName, role string, teacherUID int64) (int64, error) {
	lessonID := m.Node.Generate().Int64()
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		lesson := model.WebrtcLesson{
			LessonId:    lessonID,
			Name:        name,
			Description: desc,
			TeacherName: teacherName,
			TeacherUID:  teacherUID,
		}
		if err := tx.Create(&lesson).Error; err != nil {
			return err
		}

		ls := model.LessonStudent{
			LessonID: lesson.LessonId,
			UserID:   teacherUID,
			Role:     role,
			Status:   1,
		}

		if err := tx.Create(&ls).Error; err != nil {
			return err
		}

		payload := map[string]any{
			"lessonId": lessonID,
		}

		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		outbox := model.OutboxEvent{
			AggregateType: "Lesson",
			AggregateID:   strconv.FormatInt(lessonID, 10),
			Type:          "ADD_BLOOM",
			Payload:       string(b),
		}

		return tx.Create(&outbox).Error
	})
	if err != nil {
		return 0, err
	}
	return lessonID, nil
}

func (m *DBManager) DelLesson(lessonid int64) error {
	return m.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("lesson_id = ?", lessonid).
			Unscoped().
			Delete(&model.WebrtcLesson{}).Error; err != nil {
			return err
		}

		if err := tx.Where("lesson_id = ?", lessonid).
			Unscoped().
			Delete(&model.LessonStudent{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (m *DBManager) SelectLesson(lessonid int64) (*model.WebrtcLesson, error) {
	var lesson model.WebrtcLesson
	if err := m.DB.Where("lesson_id = ?", lessonid).First(&lesson).Error; err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (m *DBManager) SelectLessonByNandT(name, teacherName string) (*model.WebrtcLesson, error) {
	var lesson model.WebrtcLesson
	if err := m.DB.Where("name = ? AND teacher_name = ?", name, teacherName).First(&lesson).Error; err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (m *DBManager) ChangeUserToLesson(lessonid int64, stuid int64, options string) error {
	if options != "add" && options != "del" {
		return errors.New("invalid options")
	}

	return m.DB.Transaction(func(tx *gorm.DB) error {
		var lesson model.WebrtcLesson
		if err := tx.Where("lesson_id = ?", lessonid).First(&lesson).Error; err != nil {
			return err
		}

		var ls model.LessonStudent
		err := tx.Where("lesson_id = ? AND user_id = ?", lessonid, stuid).First(&ls).Error

		if options == "add" {
			if err == nil {
				return tx.Model(&model.LessonStudent{}).
					Where("lesson_id = ? AND user_id = ?", lessonid, stuid).
					Update("status", 1).Error
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			return tx.Create(&model.LessonStudent{
				LessonID: lessonid,
				UserID:   stuid,
				Status:   1,
			}).Error
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.Model(&model.LessonStudent{}).
			Where("lesson_id = ? AND user_id = ?", lessonid, stuid).
			Update("status", 0).Error
	})
}

func (m *DBManager) CreateSignIn(lessonId int64, alluserid []int64, closeTime time.Time) error {
	return m.DB.Transaction(func(tx *gorm.DB) error {
		var exist model.SignIn
		err := tx.Where("lesson_id = ?", lessonId).First(&exist).Error
		if err == nil {
			return errors.New("你已创建过签到")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		s := model.SignIn{
			LessonId:      lessonId,
			AllUserId:     alluserid,
			AlreadyUserId: []int64{},
			CloseTime:     closeTime,
		}
		return tx.Create(&s).Error
	})
}

func (m *DBManager) StuSignIn(lessonId, userId int64, timeNow time.Time) (string, error) {
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		var signIn model.SignIn
		if err := tx.Where("lesson_id = ?", lessonId).First(&signIn).Error; err != nil {
			return err
		}

		if !signIn.CloseTime.IsZero() && !timeNow.Before(signIn.CloseTime) {
			if err := tx.Unscoped().Delete(&model.SignIn{}, "lesson_id = ?", lessonId).Error; err != nil {
				return err
			}
			return errors.New("close")
		}

		for _, id := range signIn.AlreadyUserId {
			if id == userId {
				return errors.New("你已经签过到了")
			}
		}

		signIn.AlreadyUserId = append(signIn.AlreadyUserId, userId)
		return tx.Save(&signIn).Error
	})

	if err != nil {
		return "", err
	}
	return "success", nil
}

func (m *DBManager) SelectSignIn(lessonid int64) (string, error) {
	var s model.SignIn
	if err := m.DB.Where("lesson_id = ?", lessonid).First(&s).Error; err != nil {
		return "", err
	}

	signed := make(map[int64]struct{}, len(s.AlreadyUserId))
	for _, uid := range s.AlreadyUserId {
		signed[uid] = struct{}{}
	}

	var notSigned []int64
	for _, uid := range s.AllUserId {
		if _, ok := signed[uid]; !ok {
			notSigned = append(notSigned, uid)
		}
	}

	var a, b strings.Builder
	for i, uid := range s.AlreadyUserId {
		if i > 0 {
			a.WriteByte('/')
		}
		a.WriteString(fmt.Sprintf("%d", uid))
	}
	for i, uid := range notSigned {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(fmt.Sprintf("%d", uid))
	}

	return fmt.Sprintf("已签到为%v, 未签到为%v", a.String(), b.String()), nil
}
func (m *DBManager) RemoveSignIn(lessonId int64) error {
	return m.DB.Where("lesson_id = ?", lessonId).Unscoped().Delete(&model.SignIn{}).Error
}

func (m *DBManager) SaveWhiteBoard(lessonid int64, file string) error {
	var doc model.ExcalidrawDoc

	err := json.Unmarshal([]byte(file), &doc)
	if err != nil {
		return err
	}

	doc.LessonId = lessonid

	err = m.DB.Create(&doc).Error
	if err != nil {
		return err
	}
	return nil
}

func (m *DBManager) GetWhiteBoardNew(lessonid int64) (*model.ExcalidrawDoc, error) {
	var docs model.ExcalidrawDoc

	err := m.DB.Where("lesson_id = ?", lessonid).First(&docs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &docs, nil
}

func (m *DBManager) IsStudentInLesson(lessonID, userID int64) (bool, error) {
	var cnt int64
	err := m.DB.Model(&model.LessonStudent{}).
		Where("lesson_id = ? AND user_id = ? AND status = 1", lessonID, userID).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (m *DBManager) ListLessonStudentIDs(lessonID int64) ([]int64, error) {
	const batchSize = 300

	type row struct {
		ID     int64 `gorm:"column:id"`
		UserID int64 `gorm:"column:user_id"`
	}

	var (
		cursorID int64
		all      = make([]int64, 0, 64)
	)

	for {
		var rows []row
		err := m.DB.Model(&model.LessonStudent{}).
			Select("id, user_id").
			Where("lesson_id = ? AND status = 1 AND id > ?", lessonID, cursorID).
			Order("id ASC").
			Limit(batchSize).
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}

		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			all = append(all, r.UserID)
			cursorID = r.ID
		}

		if len(rows) < batchSize {
			break
		}
	}

	return all, nil
}

func (m *DBManager) LessonStudentCursorPage(lessonID int64, cursorID int64, limit int) (userIDs []int64, nextCursor int64, hasMore bool, err error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	type row struct {
		ID     int64 `gorm:"column:id"`
		UserID int64 `gorm:"column:user_id"`
	}

	rows := make([]row, 0, limit+1)
	err = m.DB.Model(&model.LessonStudent{}).
		Select("id, user_id").
		Where("lesson_id = ? AND status = 1 AND id > ?", lessonID, cursorID).
		Order("id ASC").
		Limit(limit + 1).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, false, err
	}

	if len(rows) == 0 {
		return []int64{}, 0, false, nil
	}

	if len(rows) > limit {
		hasMore = true
		rows = rows[:limit]
	}

	userIDs = make([]int64, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
		nextCursor = r.ID
	}

	return userIDs, nextCursor, hasMore, nil
}

func GetAllLessonIDs(db *gorm.DB) ([]int64, error) {
	const batchSize = 300

	type row struct {
		LessonID int64 `gorm:"column:lesson_id"`
	}

	var (
		cursor int64
		all    = make([]int64, 0, 128)
	)

	for {
		var rows []row
		err := db.Model(&model.WebrtcLesson{}).
			Select("lesson_id").
			Where("lesson_id > ?", cursor).
			Order("lesson_id ASC").
			Limit(batchSize).
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}

		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			all = append(all, r.LessonID)
			cursor = r.LessonID
		}

		if len(rows) < batchSize {
			break
		}
	}

	return all, nil
}
