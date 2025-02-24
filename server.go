package main

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

func NewServer(port string, ips chan string, sem chan bool) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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
	})

	return &http.Server{
		Addr:    ":" + port,
		Handler: handler,
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
}
