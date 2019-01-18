package main

import (
	"bufio"
	"github.com/arkadybag/golang-proxy/dao"
	"github.com/arkadybag/golang-proxy/model"
	"github.com/google/tcpproxy"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	ln, err := net.Listen("tcp", ":"+port)

	log.Println("SERVER START ON PORT:", port)
	log.Println("TIME START:", time.Now())

	log.Println("")
	if err != nil {
		log.Println("local address can not connect:", port, err.Error())
	}

	ips := make(chan string, 50)
	go getProxyUrl(ips)

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("local can not accept connect:", err.Error())
		}
		go serveConn(c, <-ips)
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

func getProxyUrl(ips chan string) {
	for {
		query := "select * from proxy order by score desc limit 50"

		proxy, err := dao.GetSQLResult("proxy", query)
		if err != nil {
			getProxyUrl(ips)
		}

		proxyModels := *proxy.(*[]model.Proxy)

		for _, proxyModel := range proxyModels {
			ips <- proxyModel.Content
		}
	}
}
