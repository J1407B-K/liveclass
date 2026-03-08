package dao

import (
	"encoding/json"
	"errors"
	"liveclass/internal/rpc/user/model"
	"strconv"

	"gorm.io/gorm"
)

func (m *DBManager) CreateUser(p model.RegisterParam) (int64, error) {
	userid := m.Node.Generate().Int64()
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		u := model.User{
			UserID:       userid,
			Username:     p.Username,
			PasswordHash: p.PasswordHash,
			Auth:         p.Auth,
			Status:       p.Status,
		}

		err := tx.Create(&u).Error
		if err != nil {
			return err
		}

		payload := map[string]any{
			"userid": userid,
		}

		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		outbox := model.OutboxEvent{
			AggregateType: "User",
			AggregateID:   strconv.FormatInt(userid, 10),
			Type:          "ADD_BLOOM",
			Payload:       string(b),
		}

		return tx.Create(&outbox).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return 0, errors.New("user already exists")
		}
		return 0, err
	}

	return userid, nil
}

func (m *DBManager) SelectUser(k int64) (*model.User, error) {
	var u model.User

	err := m.DB.First(&u, k).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *DBManager) SelectUserByUsername(username string) (*model.User, error) {
	var u model.User

	err := m.DB.Select("user_id", "username", "password_hash", "auth", "status").
		Where("username = ?", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetAllUserIDs(db *gorm.DB) ([]int64, error) {
	const batchSize = 300

	type row struct {
		UserID int64 `gorm:"column:user_id"`
	}

	var (
		cursor int64
		all    = make([]int64, 0, 128)
	)

	for {
		var rows []row

		err := db.Model(&model.User{}).
			Select("user_id").
			Where("user_id > ?", cursor).
			Order("user_id ASC").
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
			cursor = r.UserID
		}

		if len(rows) < batchSize {
			break
		}
	}

	return all, nil
}
