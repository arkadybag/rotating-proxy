package main

import (
	"bufio"
	"github.com/google/tcpproxy"
	"github.com/jinzhu/gorm"
	"log"
	"net"
	"os"
	"proxy-miner/models"
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

	ips := make(chan string, 100)
	go getProxyUrl(ips, db)

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Println("local can not accept connect:", err.Error())
		}
		go serveConn(c, ips)
	}
}

func serveConn(c net.Conn, ips chan string) {
	br := bufio.NewReader(c)
	target := tcpproxy.To(<-ips)

	if n := br.Buffered(); n > 0 {
		peeked, _ := br.Peek(br.Buffered())
		c = &tcpproxy.Conn{
			Peeked: peeked,
			Conn:   c,
		}
	}
	log.Println("handle for:", c.LocalAddr(), c.RemoteAddr())
	target.HandleConn(c)
	_ = c.Close()
}

func getProxyUrl(ips chan string, db *gorm.DB) {
	for {
		proxies := []*models.Proxy{}

		db.Table("proxies").
			Select("content").
			Where("update_time <= ?", time.Now().Unix()-int64(240)).
			Order("score desc").
			Limit(100).
			Find(&proxies)

		for _, proxy := range proxies {
			ips <- proxy.Content
		}
	}
}
