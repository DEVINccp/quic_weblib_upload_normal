package main

import (
	"bytes"
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"io"
	"log"
	"net/http"
	"sync"
)

func main() {
	urls := "https://127.0.0.1:6121/demo/getInfo"
	client := http.Client{
		Transport: &http3.RoundTripper{},
	}

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for range urls {
		addr := "https://127.0.0.1:6121/demo/getInfo"
		fmt.Println(addr)
		go func(addr string) {
			rsp, err := client.Get(addr)
			if err != nil {
				log.Fatal(err)
			}
			body := &bytes.Buffer{}
			_, err = io.Copy(body, rsp.Body)
			fmt.Println(body)
			wg.Done()
		}(string(addr))
		wg.Wait()
	}

}