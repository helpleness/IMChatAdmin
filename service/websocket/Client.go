package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"runtime/debug"
	"time"
)

type Client struct {
	Addr          string
	Socket        *websocket.Conn
	send          chan []byte
	AppID         uint32
	UserID        uint
	FirstTime     time.Time
	HeartbeatTime time.Time
	LoginTime     time.Time
}

func CreateClient(addr string, socket *websocket.Conn) *Client {
	return &Client{
		Addr:          addr,
		Socket:        socket,
		FirstTime:     time.Now(),
		HeartbeatTime: time.Now(),
		LoginTime:     time.Now(),
	}
}

func (c *Client) Read() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("write stop", string(debug.Stack()), err)
		}
	}()
	defer func() {
		fmt.Println("读取客户数据 关闭send", c)
		close(c.send)
	}()
	for {
		_, message, err := c.Socket.ReadMessage()
		if err != nil {
			fmt.Println("读取客户端数据 错误", c.Addr, err.Error())
			return
		}
		fmt.Println("读取客户端数据：", string(message))

	}
}

func (c *Client) Write() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("write stop", string(debug.Stack()), err)
		}
	}()
	defer func() {
		clientManager.Disconnect <- c
		_ = c.Socket.Close()
		fmt.Println("Client发送数据 defer", c)
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				fmt.Println("Client发送数据 关闭连接", c.Addr, "ok", ok)
				return
			}
			err := c.Socket.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				panic(err)
			}
		}
	}
}
