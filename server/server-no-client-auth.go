package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, HTTPS!")
	})

	// サーバー証明書と秘密鍵を指定してHTTPSサーバーを起動
	log.Println("Starting HTTPS server locahost:8443 ...")
	err := http.ListenAndServeTLS("localhost:8443", "server.crt", "server.key", nil)
	if err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}
}
