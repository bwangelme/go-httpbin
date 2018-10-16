package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func IPHandler(w http.ResponseWriter, r *http.Request) {
	ip := w.Header().Get("X-Forwarded-For")
	if len(ip) == 0 {
		ip = r.RemoteAddr
	}

	js, err := json.Marshal(struct {
		IP string
	}{
		IP: ip,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func Base64Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		log.Fatalln("INVALID encoded val")
		return
	}

	decodedVal, err := base64.StdEncoding.DecodeString(vars["value"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(decodedVal))
}

func main() {
	r := mux.NewRouter()
	get_router := r.Methods([]string{"GET", "HEAD"}...).Subrouter()
	get_post_router := r.Methods([]string{"GET", "HEAD", "POST"}...).Subrouter()


	get_router.HandleFunc("/ip", IPHandler)
	get_post_router.HandleFunc("/base64/{value}", Base64Handler)

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	log.Fatal(http.ListenAndServe("localhost:8080", loggedRouter))
}
