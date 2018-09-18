
package main

import (
	. "kiwi/src/server"
)


func main() {
	InitServer()
	StartServer()
	WaitServerClose()
	CloseServer()
}

// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//
//package main
//
//import (
//	"flag"
//	"fmt"
//	"log"
//	"strings"
//
//	"kiwi/src/evio"
//	"io"
//)
//
//func main() {
//	var port int
//	var loops int
//	var udp bool
//	var trace bool
//	var reuseport bool
//	var stdlib bool
//
//	flag.IntVar(&port, "port", 5000, "server port")
//	flag.BoolVar(&udp, "udp", false, "listen on udp")
//	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
//	flag.BoolVar(&trace, "trace", false, "print packets to console")
//	flag.IntVar(&loops, "loops", 0, "num loops")
//	flag.BoolVar(&stdlib, "stdlib", false, "use stdlib")
//	flag.Parse()
//
//	var events evio.Events
//	events.NumLoops = loops
//	events.Opened = func(c evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
//		fmt.Println("Opened")
//		return
//	}
//	events.Detached = func(c evio.Conn, rwc io.ReadWriteCloser) (action evio.Action) {
//		fmt.Println("Detached")
//		return
//	}
//	events.Serving = func(esr evio.EvioServer) (action evio.Action) {
//		log.Printf("echo server started on port %d (loops: %d)", port, esr.NumLoops)
//		if reuseport {
//			log.Printf("reuseport")
//		}
//		if stdlib {
//			log.Printf("stdlib")
//		}
//		return
//	}
//	events.Closed = func(c evio.Conn, err error) (action evio.Action) {
//		fmt.Println("conn closed!")
//		return
//	}
//	events.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
//		if trace {
//			log.Printf("%s", strings.TrimSpace(string(in)))
//		}
//		fmt.Println(len(in))
//		out = in
//		return
//	}
//	scheme := "tcp"
//	if udp {
//		scheme = "udp"
//	}
//	if stdlib {
//		scheme += "-net"
//	}
//	log.Fatal(evio.Serve(events, fmt.Sprintf("%s://:%d?reuseport=%t", scheme, port, reuseport)))
//}
//
