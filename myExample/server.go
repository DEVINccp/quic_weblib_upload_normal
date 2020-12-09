package main

import (
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"io"
	"net/http"
)

type tracingHandler struct {
	handler http.Handler
}


func main() {
	certPem := "/home/chengpingcai/Devlop/soucode/quic-go-0.18.1/internal/testdata/cert.pem"
	privKey := "/home/chengpingcai/Devlop/soucode/quic-go-0.18.1/internal/testdata/priv.key"
	handler := setupHandler()
	httpHandler := httpHandler()
	http.ListenAndServeTLS("127.0.0.1:443",certPem,privKey,httpHandler)
	http3.ListenAndServeQUIC("127.0.0.1:443",certPem,privKey,handler)
}

func httpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Alt-Svc","h3-29=\":443\"; ma=2592000, quic=\":443\"; ma=2592000")
		writer.Header().Add("Alternate-Protocol","h3-29=\":443\"; ma=2592000, quic=\":443\"; ma=2592000")
		io.WriteString(writer, `<html><body><form action="/demo/tiles" method="get"><a href="https://127.0.0.1:443/demo/tiles">visit demoTiles</a></body></html>`)
	})
	return &tracingHandler{handler: mux}
}

func setupHandler() http.Handler {
	wwwDir := "/home/chengpingcai/Devlop/soucode/quic-go-0.18.1"
	mux := http.NewServeMux()
	mux.Handle("/",http.FileServer(http.Dir(wwwDir)))
	mux.HandleFunc("/demo/tile", func(w http.ResponseWriter, r *http.Request) {
		// Small 40x40 png
		w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x28,
			0x01, 0x03, 0x00, 0x00, 0x00, 0xb6, 0x30, 0x2a, 0x2e, 0x00, 0x00, 0x00,
			0x03, 0x50, 0x4c, 0x54, 0x45, 0x5a, 0xc3, 0x5a, 0xad, 0x38, 0xaa, 0xdb,
			0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41, 0x54, 0x78, 0x01, 0x63, 0x18,
			0x61, 0x00, 0x00, 0x00, 0xf0, 0x00, 0x01, 0xe2, 0xb8, 0x75, 0x22, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		})
	})

	mux.HandleFunc("/demo/tiles", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><head><style>img{width:40px;height:40px;}</style></head><body>")
		for i := 0; i < 200; i++ {
			fmt.Fprintf(w, `<img src="/demo/tile?cachebust=%d">`, i)
		}
		io.WriteString(w, "</body></html>")
	})

	mux.HandleFunc("/demo/getInfo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello QUIC"))
	})

	return &tracingHandler{handler: mux}
}

func (h *tracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}