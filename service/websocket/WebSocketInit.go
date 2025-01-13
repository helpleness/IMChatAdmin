package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"net/http"
)

const (
	defaultAppID = 101 // 默认平台ID
)

var (
	clientManager = NewClientManager()                    // 管理者
	appIDs        = []uint32{defaultAppID, 102, 103, 104} // 全部的平台
	serverIp      string
	serverPort    string
)

func SocketStart() {
	defer func() {
		if err := recover(); err != nil {
		}
	}()
	serverIp = viper.GetString("websocket.ip")
	serverPort = viper.GetString("websocket.rpcPort")
	WebSocketPort := viper.GetString("websocket.port")
	http.HandleFunc("/ws/default.io", wsPage)
	fmt.Println("WebSocket 启动程序成功", serverIp, serverPort)
	err := http.ListenAndServe(":"+WebSocketPort, nil)
	if err != nil {
		panic(err)
	}
}
func wsPage(w http.ResponseWriter, r *http.Request) {
	// 将 HTTP 请求升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade to websocket:", err)
		return
	}
	fmt.Println("Client connected!", conn.RemoteAddr().String())
	client := CreateClient(conn.RemoteAddr().String(), conn)
	go client.Read()
	go client.Write()
	clientManager.Connect <- client
}

// 定义 WebSocket 升级器
var upgrader = websocket.Upgrader{
	// 允许跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
