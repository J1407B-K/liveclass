package global

import (
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

var (
	//websocket升级器
	Upgrader = websocket.HertzUpgrader{
		CheckOrigin: func(c *app.RequestContext) bool {
			return true
		},
	}

	//储存连接的map
	WsConnsQuiz       = make(map[*websocket.Conn]int64) //userid
	WsConnsQuizLesson = make(map[*websocket.Conn]int64) //lessonid
	WsConnsChat       = make(map[*websocket.Conn]int64)
	//锁
	Mux = sync.Mutex{}

	KafkaBroker = "127.0.0.1:9092"
	KafkaTopic  = "local-dev"
)
