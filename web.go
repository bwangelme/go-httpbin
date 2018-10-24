package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
)

var (
	CWD          string
	TEMPLATE_DIR string
	STATIC_DIR   string
)

func init() {
	CWD, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalln(err)
	}
	TEMPLATE_DIR = filepath.Join(CWD, "templates")
	STATIC_DIR = filepath.Join(CWD, "static")
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	index_tmpl, err := template.ParseGlob(filepath.Join(TEMPLATE_DIR, "*"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	index_tmpl.Execute(w, nil)
	return
}

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
		log.Fatalln("INVALID path")
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

func registerHandler() http.Handler {
	var r = mux.NewRouter()
	get_router := r.Methods([]string{"GET", "HEAD"}...).Subrouter()
	get_post_router := r.Methods([]string{"GET", "HEAD", "POST"}...).Subrouter()

	r.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))))
	get_router.HandleFunc("/legacy", IndexHandler)
	get_router.HandleFunc("/ip", IPHandler)
	get_post_router.HandleFunc("/base64/{value}", Base64Handler)

	handler := handlers.LoggingHandler(os.Stdout, r)

	return handler
}

func main() {
	var handler = registerHandler()
	var wait time.Duration
	flag.DurationVar(&wait, "shutdownTime", 15*time.Second, "服务器被关闭时的等待时间")
	flag.Parse()

	srv := &http.Server{
		Addr:         "localhost:8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      handler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)
	log.Println("Graceful shutdown the server")

	os.Exit(0)
}
