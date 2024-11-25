package server

import "net/http"

func New() {
	server := http.Server{}

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server.ListenAndServe()
}
