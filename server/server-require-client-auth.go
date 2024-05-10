package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			// クライアントの証明書を取得
			clientCert := r.TLS.PeerCertificates[0]
			fmt.Println("Client Certificate Subject:", clientCert.Subject)
		}
		w.Write([]byte("Hello, HTTPS!"))
	})

	// サーバーの証明書と秘密鍵を読み込む
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificates: %v", err)
	}

	// 認証局の証明書を読み込む
	caCert, err := os.ReadFile("../client-ca/client-ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// 証明書認証を要求するTLS設定を作成
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}

	// HTTPSサーバーを設定
	server := &http.Server{
		Addr:      "localhost:9443",
		TLSConfig: tlsConfig,
	}

	// HTTPSサーバーを起動
	log.Println("Starting HTTPS server locahost:9443 ...")
	err = server.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatalf("Failed to start HTTPS server: %v", err)
	}
}
