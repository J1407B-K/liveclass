package initialize

import (
	"context"
	"github.com/go-redis/redis/v8"
	"os"
)

func InitScript(rdb *redis.Client) (string, string, string, string) {
	//读取lua脚本
	countscriptb, err := os.ReadFile("./rpc/live/lua/live_room_count.lua")
	if err != nil {
		panic(err)
	}

	memberscriptb, err := os.ReadFile("./rpc/live/lua/live_room_member.lua")
	if err != nil {
		panic(err)
	}

	deletescriptb, err := os.ReadFile("./rpc/live/lua/del_room_allkey.lua")
	if err != nil {
		panic(err)
	}

	selectscriptb, err := os.ReadFile("./rpc/live/lua/select_room_info.lua")
	if err != nil {
		panic(err)
	}

	countscript := string(countscriptb)
	memberscript := string(memberscriptb)
	deletescript := string(deletescriptb)
	selectscript := string(selectscriptb)

	//加载到redis缓存
	countsha, err := rdb.ScriptLoad(context.Background(), countscript).Result()
	if err != nil {
		panic(err)
	}
	membersha, err := rdb.ScriptLoad(context.Background(), memberscript).Result()
	if err != nil {
		panic(err)
	}
	deletesha, err := rdb.ScriptLoad(context.Background(), deletescript).Result()
	if err != nil {
		panic(err)
	}
	selectsha, err := rdb.ScriptLoad(context.Background(), selectscript).Result()
	if err != nil {
		panic(err)
	}

	return countsha, membersha, deletesha, selectsha
}
