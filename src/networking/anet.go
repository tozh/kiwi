package networking

import (
	"fmt"
	"net"
	"time"
	. "redigo/src/constant"
)

func AnetSetErrorFormat(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a)
}

func AnetSetTcpKeepALive(conn *net.TCPConn, keepalive bool) int64 {
	if err := conn.SetKeepAlive(keepalive); err != nil {
		AnetSetErrorFormat("Set tcp KeepAlive ---> %t, error: %s", keepalive, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTcpNoDelay(conn *net.TCPConn, noDelay bool) int64 {
	if err := conn.SetNoDelay(noDelay); err != nil {
		AnetSetErrorFormat("Set tcp NoDelay ---> %t, error: %s", noDelay, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTimeout(conn *net.TCPConn, timeMs int64) int64 {
	if err := conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeMs))); err != nil {
		AnetSetErrorFormat("Set Timeout(ms) ---> %d, error: %s", timeMs, err)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetTcpAddress(ip string, port int64) string {
	return fmt.Sprintf("%s:%d", ip, port)
}

func AnetListenUnix(address string) *net.UnixListener {
	addr, err1 := net.ResolveUnixAddr("unix", address)
	if err1 != nil {
		return nil
	}
	listener, err2 := net.ListenUnix("unix", addr)
	if err2 != nil {
		AnetSetErrorFormat("Listen err2: %s", err2)
		return nil
	}
	return listener
}

func AnetListenTcp(tcpType string, ip string, port int64) *net.TCPListener {
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

//func AnetAccept(listener net.Listener) net.Conn {
//	for {
//		conn, err := listener.Accept()
//		if err != nil {
//			AnetSetErrorFormat("Accept error: %s", error)
//			continue
//		}
//		//会打断for 应该让for一直存在
//		//这块代码重写
//		return conn
//	}
//}
//
//func AnetTcpServer(tcpType string, ip string, port int64) net.Conn {
//	// tcpType: "tcp4" or "tcp6"
//	listener := AnetListenTcp(tcpType, ip, port)
//	if listener == nil {
//		return nil
//	}
//	return AnetAccept(listener)
//}
//
//func AnetUnixServer(address string) net.Conn {
//	listener := AnetListenUnix(address)
//	if listener == nil {
//		return nil
//	}
//	return AnetAccept(listener)
//}
