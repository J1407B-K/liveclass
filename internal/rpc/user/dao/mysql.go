package dao

import (
	"errors"
	"liveclass/internal/rpc/user/model"

	"gorm.io/gorm"
)

func (m *DBManager) SaveUser(p model.RegisterParam) (int64, error) {
	sid := m.Node.Generate().Int64()
	u := model.User{
		UserID:       sid,
		Username:     p.Username,
		PasswordHash: p.PasswordHash,
		Auth:         p.Auth,
		Status:       p.Status,
	}

	err := m.DB.Create(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return 0, errors.New("user already exists")
		}
		return 0, err
	}
	return sid, nil
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
