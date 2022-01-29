package main

import (
	"log"
	"net/http"

	"github.com/ysmood/gate/lib/client"
)

func main() {
	l, err := client.New("test.com", "x", "xxx")
	if err != nil {
		log.Fatal(err)
	}

	http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello!"))
	}))
}
