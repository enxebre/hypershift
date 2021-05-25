package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ign/{nodePool}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("http://%s-mcs.svc", mux.Vars(r)["nodePool"]), 301)
	})

	if err := http.ListenAndServe(":9090", r); err != nil {
		panic(err)
	}
}
