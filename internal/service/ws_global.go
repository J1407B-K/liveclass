package service

import (
	jwtt "github.com/golang-jwt/jwt/v4"
	"liveclass/internal/global"
)

var jwtSecret = []byte("by_kq")

type Claims struct {
	UserId string `json:"userId"`
	jwtt.RegisteredClaims
}

func parse(tokenStr string) (*Claims, error) {
	token, err := jwtt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}

// 传入教师userid，教师端统计
func broadcastToTeacher(userid string, message interface{}) error {
	global.Mux.Lock()
	defer global.Mux.Unlock()
	for conn, uid := range global.WsConnsQuiz {
		if uid == userid {
			err := conn.WriteJSON(message)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
