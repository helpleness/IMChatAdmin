package controller

import (
	"encoding/json"
	"github.com/helpleness/IMChatAdmin/model"
	"github.com/helpleness/IMChatAdmin/service/websocket"
)

func LoginController(client *websocket.Client, seq string, message []byte) (code uint32, msg string, data interface{}) {
	code = 200
	//currentTime := uint32(time.Now().Unix())
	request := &model.User{}
	if err := json.Unmarshal(message, request); err != nil {

	}

	return
}
