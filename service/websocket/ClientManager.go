package websocket

import "sync"

type ClientManager struct {
	Client      map[*Client]bool   //全部的连接
	ClientsLock sync.RWMutex       //读写锁
	Users       map[string]*Client //登陆的用户 //appID+uuid
	UsersLock   sync.RWMutex       //读写锁
	Connect     chan *Client       //连接处理
	Disconnect  chan *Client       //断开连接处理
	Login       chan *Client       //用户登录处理
	Broadcast   chan *Client       //广播 向全部成员发送数据
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Client:     make(map[*Client]bool),
		Users:      make(map[string]*Client),
		Connect:    make(chan *Client),
		Disconnect: make(chan *Client),
		Login:      make(chan *Client),
		Broadcast:  make(chan *Client),
	}
}
