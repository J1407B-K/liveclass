package global

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

var (
	//websocket升级器
	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	//储存连接的map
	WsConns = make(map[*websocket.Conn]bool)
	//锁
	Mux = sync.Mutex{}
)
