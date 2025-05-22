package main

import (
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/internal/rpc/agent/mcp"
	"log"
)

func main() {
	go mcp.StartMCPServer()

	svr := agent.NewServer(new(AgentServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
