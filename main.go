package main

import (
	"bufio"
	"github.com/google/tcpproxy"
	"log"
	"myProxy/rotateProxy"
	"net"
)

func main() {
	proxyServer := rotateProxy.NewProxyServer()
	proxyServer.Init()

	localAddr := ":8080"
	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Println("local address can not connect:", localAddr, err.Error())
	}

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("local can not accept connect", err.Error())
		}
		go serveConn(c, proxyServer.GetProxy())
	}
}

func serveConn(c net.Conn, proxyIpPort string) {
	br := bufio.NewReader(c)
	target := tcpproxy.To(proxyIpPort)

	if n := br.Buffered(); n > 0 {
		peeked, _ := br.Peek(br.Buffered())
		c = &tcpproxy.Conn{
			Peeked: peeked,
			Conn:   c,
		}
	}
	log.Println("handle for:", c.LocalAddr(), c.RemoteAddr())
	target.HandleConn(c)
	c.Close()
}
