package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"proxy-miner/models"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := NewPostgreSQL()
	defer db.Close()

	if err != nil {
		log.Fatalln("can not connect to postgres:", err)
	}

	ips := make(chan string, 200)
	sem := make(chan bool, 200)
	go cleaner(ips)

	go getProxyUrl(ips, db)

	server := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			auth := strings.SplitN(r.Header.Get("Proxy-Authorization"), " ", 2)

			if len(auth) != 2 || auth[0] != "Basic" {
				http.Error(w, "authorization failed", http.StatusUnauthorized)
				return
			}

			payload, _ := base64.StdEncoding.DecodeString(auth[1])
			pair := strings.SplitN(string(payload), ":", 2)

			if len(pair) != 2 || pair[0] != os.Getenv("proxy_username") && pair[1] != os.Getenv("proxy_password") {
				http.Error(w, "authorization failed", http.StatusUnauthorized)
				return
			}

			r.Header.Del("Proxy-Authorization")

			sem <- true
			if r.Method == http.MethodConnect {
				handleTunneling(w, r, ips, sem)
			} else {
				handleHTTP(w, r, ips, sem)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Println("SERVER START ON PORT:", port)
	log.Println("TIME START:", time.Now())

	log.Fatal(server.ListenAndServe())
}

func cleaner(ips chan string) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for range ticker.C {
		<-ips
		log.Println("cleaner TICK")
	}
}

func handleTunneling(w http.ResponseWriter, r *http.Request, ips chan string, sem chan bool) {
	defer func() {
		<-sem
	}()
	var counter uint64

	dest_conn, err := dialCoordinatorViaCONNECT(r.Host, ips, &counter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)

}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	_, _ = io.Copy(destination, source)
}

func dialCoordinatorViaCONNECT(addr string, ips chan string, counter *uint64) (net.Conn, error) {
	if *counter > 5 {
		err := fmt.Sprintf("error with max counter retry for: %s", addr)
		log.Printf(err)
		return nil, errors.New(err)
	}
	atomic.AddUint64(counter, 1)

	proxyAddr := <-ips

	log.Printf("dialing proxy %q to remote: %s", proxyAddr, addr)
	c, err := net.DialTimeout("tcp", proxyAddr, time.Second*10)
	if err != nil {
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	err = c.SetReadDeadline(time.Now().Add(time.Second * 15))

	if err != nil {
		c.Close()
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}

	_, err = fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", addr, addr)
	if err != nil {
		c.Close()

		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	br := bufio.NewReader(c)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		c.Close()

		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	if res.StatusCode != 200 {
		c.Close()

		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}

	if br.Buffered() > 0 {
		c.Close()

		log.Printf("unexpected %d bytes of buffered data from CONNECT proxy %q",
			br.Buffered(), proxyAddr)
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	return c, nil
}

func handleHTTP(w http.ResponseWriter, req *http.Request, ips chan string, sem chan bool) {
	defer func() {
		<-sem
	}()
	execHandleHTTP(w, req, ips)
}

func execHandleHTTP(w http.ResponseWriter, req *http.Request, ips chan string) {
	proxyUrl, err := url.Parse(fmt.Sprintf("http://%s", <-ips))
	myClient := &http.Transport{Proxy: http.ProxyURL(proxyUrl)}

	resp, err := myClient.RoundTrip(req)

	if err != nil {
		execHandleHTTP(w, req, ips)
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func getProxyUrl(ips chan string, db *gorm.DB) {
	for {
		proxies := []*models.Proxy{}

		db.Table("proxies").
			Select("content").
			Order("score desc").
			Limit(200).
			Find(&proxies)

		for _, proxy := range proxies {
			ips <- proxy.Content
		}
	}
}
