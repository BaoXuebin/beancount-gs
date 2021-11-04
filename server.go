package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	route := gin.Default()
	route.StaticFS("/", http.Dir("./public"))
	_ = http.ListenAndServe(":3001", nil)
}
