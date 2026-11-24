package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("APP READY")
	fmt.Println(http.ListenAndServe("127.0.0.1:8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})))
}
