package server

import (
	"fmt"
	"net"
	"time"
)

func AnetSetErrorFormat(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a)
}

func AnetSetTcpKeepALive(conn *net.TCPConn, keepalive bool) int {
	if err := conn.SetKeepAlive(keepalive); err != nil {
		AnetSetErrorFormat("Set tcp KeepAlive ---> %t, error: %s", keepalive, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTcpNoDelay(conn *net.TCPConn, noDelay bool) int {
	if err := conn.SetNoDelay(noDelay); err != nil {
		AnetSetErrorFormat("Set tcp NoDelay ---> %t, error: %s", noDelay, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTimeout(conn *net.TCPConn, timeMs int) int {
	if err := conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeMs))); err != nil {
		AnetSetErrorFormat("Set Timeout(ms) ---> %d, error: %s", timeMs, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetTcpAddress(ip string, port int) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func AnetListenUnix(address string) *net.UnixListener {
	addr, err1 := net.ResolveUnixAddr("unix", address)
	if err1 != nil {
		// fmt.Println("1 --------> ", err1)
		return nil
	}
	listener, err2 := net.ListenUnix("unix", addr)
	if err2 != nil {
		// fmt.Println("2 --------> ", err2)
		AnetSetErrorFormat("Listen err2: %s", err2)
		return nil
	}
	return listener
}

func AnetListenTcp(tcpType string, ip string, port int) *net.TCPListener {
	// tcpType: "tcp", "tcp4" or "tcp6"
	addr := AnetTcpAddress(ip, port)
	address, err1 := net.ResolveTCPAddr(tcpType, addr)
	if err1 != nil {
		return nil
	}
	listener, err2 := net.ListenTCP(tcpType, address)
	if err2 != nil {
		AnetSetErrorFormat("Listen error: %s", err2)
		return nil
	}
	return listener
}
