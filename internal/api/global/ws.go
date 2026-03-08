package global

import (
	"liveclass/internal/api/model"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

var (
	Upgrader = websocket.HertzUpgrader{
		CheckOrigin: func(c *app.RequestContext) bool {
			return true
		},
	}

	//储存连接的map
	WsConnsQuiz     = make(map[*websocket.Conn]*model.QuizConnMeta)
	ChatLessonConns = make(map[int64]map[*websocket.Conn]struct{})
	//锁
	Mux = sync.RWMutex{}

	KafkaBroker = "127.0.0.1:9092"
	KafkaTopic  = "liveclass-chat"
)
