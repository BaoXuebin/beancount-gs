package main

import (
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("./public")))
	_ = http.ListenAndServe(":3001", nil)
}
