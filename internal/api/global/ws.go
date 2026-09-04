package global

import (
	"liveclass/internal/api/chatroom"
	"liveclass/internal/api/model"
	"net/url"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

var (
	Upgrader = websocket.HertzUpgrader{
		CheckOrigin: originChecker(nil),
	}

	//储存连接的map
	WsConnsQuiz = make(map[*websocket.Conn]*model.QuizConnMeta)
	ChatRooms   *chatroom.Manager
	//锁
	Mux = sync.RWMutex{}
)

func ConfigureWebSocketUpgrader(allowedOrigins []string) {
	Upgrader.CheckOrigin = originChecker(allowedOrigins)
}

func originChecker(allowedOrigins []string) func(*app.RequestContext) bool {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return func(ctx *app.RequestContext) bool {
		origin := strings.TrimRight(strings.TrimSpace(string(ctx.Request.Header.Peek("Origin"))), "/")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return false
		}
		if strings.EqualFold(parsed.Host, string(ctx.Request.Host())) {
			return true
		}
		_, ok := allowed[origin]
		return ok
	}
}
