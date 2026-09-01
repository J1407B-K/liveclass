package initialize

import (
	"context"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

func InitScript(rdb *redis.Client) (string, string, string) {
	//读取lua脚本
	changescriptb, err := os.ReadFile("./rpc/webrtc_live/lua/live_room_change.lua")
	if err != nil {
		panic(err)
	}

	deletescriptb, err := os.ReadFile("./rpc/webrtc_live/lua/del_room_allkey.lua")
	if err != nil {
		panic(err)
	}

	selectscriptb, err := os.ReadFile("./rpc/webrtc_live/lua/select_room_info.lua")
	if err != nil {
		panic(err)
	}

	changescript := string(changescriptb)
	deletescript := string(deletescriptb)
	selectscript := string(selectscriptb)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	changesha, err := rdb.ScriptLoad(ctx, changescript).Result()
	if err != nil {
		panic(err)
	}
	deletesha, err := rdb.ScriptLoad(ctx, deletescript).Result()
	if err != nil {
		panic(err)
	}
	selectsha, err := rdb.ScriptLoad(ctx, selectscript).Result()
	if err != nil {
		panic(err)
	}

	return changesha, deletesha, selectsha
}
