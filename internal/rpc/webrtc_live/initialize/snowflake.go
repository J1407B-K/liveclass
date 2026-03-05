package initialize

import (
	"github.com/bwmarrin/snowflake"
)

const Node = 2

func InitSnowflake() (*snowflake.Node, error) {
	return snowflake.NewNode(Node)
}
