package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bwangelme/httpbin/middlewares"
	"github.com/google/uuid"
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

/*
 * ====================================
 * Helper Func
 * ====================================
 */

func getHeadersMap(header http.Header) map[string]string {
	headers := make(map[string]string)

	for k, v := range header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return headers
}

func getPeerIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if len(ip) == 0 {
		ip = r.RemoteAddr
	}

	return ip
}

func getRequestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

/*
 * ====================================
 * Request Handlers
 * ====================================
 */

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	indexTmpl, err := template.ParseGlob(filepath.Join(TEMPLATE_DIR, "*"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	indexTmpl.ExecuteTemplate(w, "index.html", nil)
	return
}

func IPHandler(w http.ResponseWriter, r *http.Request) {
	ip := getPeerIP(r)

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

func UUIDHandler(w http.ResponseWriter, r *http.Request) {
	uuidVal := uuid.New()

	js, err := json.Marshal(struct {
		UUID string
	}{
		UUID: uuidVal.String(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(js))
}

func UserAgentHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(struct {
		UserAgent string `json:"user-agent"`
	}{
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(js))
}

func HeadersHandler(w http.ResponseWriter, r *http.Request) {
	headers := getHeadersMap(r.Header)

	js, err := json.Marshal(headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(js))
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}
	queryArgs := make(map[string]string)
	for k, _ := range values {
		queryArgs[k] = values.Get(k)
	}

	result := make(map[string]interface{})

	result["origin"] = getPeerIP(r)
	result["url"] = fmt.Sprintf("%s://%s%s", getRequestScheme(r), r.Host, r.URL.RequestURI())
	result["headers"] = getHeadersMap(r.Header)
	result["args"] = queryArgs

	js, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(js))
}

func BytesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		log.Println("INVALID PATH")
		return
	}

	n, err := strconv.ParseInt(vars["n"], 10, 64)
	if err != nil {
		log.Println("INVALID path component", err)
		return
	}
	if n > 100*1024 {
		n = 100 * 1024
	}

	data := make([]byte, n)
	r.ParseForm()
	seed, err := strconv.ParseInt(r.Form.Get("seed"), 10, 64)
	if err == nil {
		r := rand.New(rand.NewSource(seed))
		r.Read(data)
	} else {
		log.Printf("INVALID SEED %s %s", r.Form.Get("seed"), err)
		rand.Read(data)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
	return
}

/*
 * ====================================
 * WebApp Init
 * ====================================
 */

func registerHandle(r *mux.Router) http.Handler {
	get_router := r.Methods([]string{"GET", "HEAD"}...).Subrouter()
	get_post_router := r.Methods([]string{"GET", "HEAD", "POST"}...).Subrouter()

	r.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))))
	get_router.HandleFunc("/legacy", IndexHandler)
	get_router.HandleFunc("/ip", IPHandler)
	get_router.HandleFunc("/uuid", UUIDHandler)
	get_router.HandleFunc("/user-agent", UserAgentHandler)
	get_router.HandleFunc("/headers", HeadersHandler)
	get_router.HandleFunc("/get", GetHandler)

	get_post_router.HandleFunc("/base64/{value}", Base64Handler)
	get_post_router.HandleFunc("/bytes/{n}", BytesHandler)

	handler := handlers.LoggingHandler(os.Stdout, r)

	return handler
}

func registerMiddleware(router *mux.Router) {
	//awm := middlewares.NewAuthMiddleware()
	//router.Use(awm.Middleware)
	router.Use(middlewares.JSONMiddleware)
}

func main() {
	var r = mux.NewRouter()
	registerMiddleware(r)
	var handler = registerHandle(r)
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
		log.Println("Start server on", srv.Addr)
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
