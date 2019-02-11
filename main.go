package main

import (
	"bufio"
	"crypto/tls"
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

	ips := make(chan string, 100)
	go cleaner(ips)

	go getProxyUrl(ips, db)

	server := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r, ips)
			} else {
				handleHTTP(w, r, ips)
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
	}
}

func handleTunneling(w http.ResponseWriter, r *http.Request, ips chan string) {
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
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)

}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
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
	c, err := net.DialTimeout("tcp", proxyAddr, time.Second*5)

	if err != nil {
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	_, err = fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", addr, proxyAddr)
	if err != nil {
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	br := bufio.NewReader(c)
	res, err := http.ReadResponse(br, nil)
	if err != nil {
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	if res.StatusCode != 200 {
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}

	if br.Buffered() > 0 {
		log.Printf("unexpected %d bytes of buffered data from CONNECT proxy %q",
			br.Buffered(), proxyAddr)
		log.Printf("Try again for %s ...", addr)
		return dialCoordinatorViaCONNECT(addr, ips, counter)
	}
	return c, nil
}

func handleHTTP(w http.ResponseWriter, req *http.Request, ips chan string) {
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
			Where("update_time >= ? AND score >= ?", time.Now().Unix()-int64(240), 10).
			Order("score").
			//Order(gorm.Expr("random()")).
			Limit(100).
			Find(&proxies)

		for _, proxy := range proxies {
			ips <- proxy.Content
		}
	}
}
