package client

import (
	"net"
	"fmt"
)

type Client struct {
	Buf []byte
	Conn net.Conn
}

func CreateClient(conn net.Conn) *Client {
	client := Client{
		Buf: make([]byte, 0),
		Conn: conn,
	}
	return &client
}


func TcpAddress(ip string, port int) string {
	return fmt.Sprintf("%s:%d", ip, port)
}


func ConnectTcp(address string, ip int) *Client {
	conn, err := net.Dial("tcp", TcpAddress(address, ip))
	if err != nil {
		panic("err")
		conn.Close()
		return nil
	}
	return CreateClient(conn)
}


func ConnectUnix(address string) *Client {
	conn, err := net.Dial("unix", address)
	if err != nil {
		panic("err")
		conn.Close()
		return nil
	}
	return CreateClient(conn)
}

type Clienter interface {
	Set(key string) (bool, error)
	MSet(keys ...string) (int64, error)
	Get(key string) (interface{}, error)
	MGet(keys ...string) ([]interface{}, error)
	SetNx(key string) (bool, error)
	MSetNx(keys ...string) (int64, error)
	SetEx(key string) (bool, error)
	Append(key string) (int64, error)
	StrLen(key string) (int64, error)
	Del(keys ...string) (int64, error)
	Select(id int64) (bool, error)
	FlushAll() (bool, error)
	RandomKey() (string, error)
	Incr(key string) (int64, error)
	Decr(key string) (int64, error)
	Exists(keys ...string) (int64, error)
}