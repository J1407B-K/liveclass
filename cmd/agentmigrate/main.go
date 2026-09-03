package main

import (
	"liveclass/internal/rpc/agent/flag"
	"liveclass/internal/rpc/agent/initialize"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitPGDB()
	flag.PGAutoMigrate(db)
}
