package server

import (
	"fmt"
	"net/http"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hello")
	w.WriteHeader(http.StatusOK)
}
