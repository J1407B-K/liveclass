package main

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	chat "liveclass/idl/kitex_gen/chat"
)

// ChatServiceImpl implements the last service interface defined in the IDL.
type ChatServiceImpl struct {
	mongoClient *mongo.Client
	mongoColl   *mongo.Collection
}

// CreateChatRoomResp implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) CreateChatRoomResp(ctx context.Context, req *chat.CreateChatRoomReq) (resp *chat.CreateChatRoomResp, err error) {
	// TODO: Your code here...
	return
}
