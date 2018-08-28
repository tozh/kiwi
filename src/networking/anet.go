package networking

import (
	"fmt"
	"net"
	"time"
	."redigo/src/constant"
)

func AnetSetErrorFormat(err *string, format string, a ...interface{}) {
	if err == nil  {
		return
	}
	*err = fmt.Sprintf(format, a)
}

//func AnetSetBlock(err *string, conn int64, nonBlock bool) int64 {
//	return ANET_OK
//}
//func AnetBlock(err *string, fd int64) int64 {
//	return AnetSetBlock(err, fd, false)
//}
//func AnetNonBlock(err *string, fd int64) int64 {
//	return AnetSetBlock(err, fd, true)
//}

func AnetSetTcpKeepALive(err *string, conn *net.TCPConn, keepalive bool) int64 {
	if errno := conn.SetKeepAlive(keepalive); errno != nil {
		AnetSetErrorFormat(err, "Set tcp KeepAlive ---> %t, error: %s", keepalive, errno)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTcpNoDelay(err *string, conn *net.TCPConn, noDelay bool)  int64{
	if errno := conn.SetNoDelay(noDelay); errno != nil {
		AnetSetErrorFormat(err, "Set tcp NoDelay ---> %t, error: %s", noDelay, errno)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetSetTimeout(err *string, conn *net.TCPConn, timeMs int64) int64 {
	if 	errno := conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeMs))); errno != nil {
		AnetSetErrorFormat(err, "Set Timeout(ms) ---> %d, error: %s", timeMs, errno)
		return ANET_ERR
	}
	return ANET_OK
}

func AnetTcpAddress(ip string, port int64) string {
	return fmt.Sprintf("%s:%v", ip, port)
}

func AnetListenUnix(err *string, address string) *net.UnixListener {
	addr, errAddr := net.ResolveUnixAddr("unix", address)
	if errAddr != nil {
		return nil
	}
	listener, errno := net.ListenUnix("unix", addr)
	if errno != nil {
		AnetSetErrorFormat(err, "Listen error: %s", errno)
		return nil
	}
	return listener
}

func AnetListenTcp(err *string, tcpType string, ip string, port int64) *net.TCPListener {
	// tcpType: "tcp4" or "tcp6"
	addr := AnetTcpAddress(ip, port)
	address, errAddr := net.ResolveTCPAddr(tcpType, addr)
	if errAddr != nil {
		return nil
	}
	listener, errno := net.ListenTCP(tcpType, address)
	if errno != nil {
		AnetSetErrorFormat(err, "Listen error: %s", errno)
		return nil
	}
	return listener
}


func AnetAccept(err *string, listener net.Listener) net.Conn {
	for {
		conn, errno := listener.Accept()
		if errno != nil {
			AnetSetErrorFormat(err, "Accept error: %s", errno)
			continue
		}
		return conn
	}
}

func AnetHandler(conn net.Conn) {

}

func AnetTcpServer(err *string, tcpType string, ip string, port int64) net.Conn {
	// tcpType: "tcp4" or "tcp6"
	listener := AnetListenTcp(err, tcpType, ip, port)
	if listener == nil {
		return nil
	}
	return AnetAccept(err, listener)
}

func AnetUnixServer(err *string, address string) net.Conn {
	listener := AnetListenUnix(err, address)
	if listener == nil {
		return nil
	}
	return AnetAccept(err, listener)
}






