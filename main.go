package main

import (
	"bufio"
	"github.com/arkadybag/golang-proxy/model"
	"github.com/google/tcpproxy"
	"github.com/jinzhu/gorm"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}
	ln, err := net.Listen("tcp", ":"+port)

	if err != nil {
		log.Println("local address can not connect:", port, err.Error())
	}

	log.Println("SERVER START ON PORT:", port)
	log.Println("TIME START:", time.Now())

	db, err := NewPostgreSQL()
	defer db.Close()

	if err != nil {
		log.Fatalln("can not connect to postgres:", err)
	}

	ips := make(chan string, 50)
	go getProxyUrl(ips, db)

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

func getProxyUrl(ips chan string, db *gorm.DB) {
	for {
		proxies := []*model.Proxy{}

		db.Order("score desc").Limit(50).Find(&proxies)

		for _, proxy := range proxies {
			ips <- proxy.Content
		}
	}
}
