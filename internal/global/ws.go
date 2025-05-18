package global

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
	"sync"
)

var (
	//websocket升级器
	Upgrader = websocket.HertzUpgrader{
		CheckOrigin: func(c *app.RequestContext) bool {
			return true
		},
	}

	//储存连接的map(string应为lessonName)
	WsConns = make(map[*websocket.Conn]string)
	//锁
	Mux = sync.Mutex{}
)
