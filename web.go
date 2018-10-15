package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func IPHandler(w http.ResponseWriter, r *http.Request) {
	ip := w.Header().Get("X-Forwarded-For")
	if len(ip) == 0 {
		ip = r.RemoteAddr
	}

	js, err := json.Marshal(struct {
		ip string
	}{
		ip: ip,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(js)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/ip", IPHandler)

	log.Fatal(http.ListenAndServe("localhost:8080", r))
}
