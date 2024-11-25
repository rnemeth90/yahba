package server

import "net/http"

func New() {
	r := mux.NewRouter
}

func testHandler(w http.ResponseWriter, r *http.Request) {

}
