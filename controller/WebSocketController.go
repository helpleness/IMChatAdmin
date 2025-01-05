package controller

import (
	"encoding/json"
	"go_test/model"
	"go_test/service/websocket"
)

func LoginController(client *websocket.Client, seq string, message []byte) (code uint32, msg string, data interface{}) {
	code = 200
	//currentTime := uint32(time.Now().Unix())
	request := &model.Login{}
	if err := json.Unmarshal(message, request); err != nil {

	}

	return
}
