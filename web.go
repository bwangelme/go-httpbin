package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	logger       = NewWebLogger()
)

func init() {
	CWD, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logger.Fatalln(err)
	}
	TEMPLATE_DIR = filepath.Join(CWD, "templates")
	STATIC_DIR = filepath.Join(CWD, "static")
}

func unescaped (x string) interface{} { return template.HTML(x) }

/*
 * ====================================
 * Request Handlers
 * ====================================
 */

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: 将模板渲染函数统一起来
	indexTmpl, err := template.ParseGlob(filepath.Join(TEMPLATE_DIR, "*"))

	indexTmpl.Funcs(template.FuncMap{"html": unescaped})
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}
	err = indexTmpl.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"URL_CONFIG": URL_CONFIG,
		"URL_GROUP_CONFIG": URL_GROUP_CONFIG,
	})
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}
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
		logger.InternalErrorPrint(w, err.Error())
		return
	}
	w.Write(js)
}

func Base64Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		logger.Fatalln("INVALID path")
		return
	}

	decodedVal, err := base64.StdEncoding.DecodeString(vars["value"])
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
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
		logger.InternalErrorPrint(w, err.Error())
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
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	fmt.Fprintf(w, string(js))
}

func HeadersHandler(w http.ResponseWriter, r *http.Request) {
	headers := getHeadersMap(r.Header)

	js, err := json.Marshal(headers)
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	fmt.Fprintf(w, string(js))
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
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
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	fmt.Fprintf(w, string(js))
}

func BytesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		logger.Println("INVALID PATH")
		return
	}

	n, err := strconv.ParseInt(vars["n"], 10, 64)
	if err != nil {
		logger.Println("INVALID path component", err)
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
		logger.Printf("INVALID SEED %s %s", r.Form.Get("seed"), err)
		rand.Read(data)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
	return
}

func StreamBytesHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.Fatalln("Excepted http.ResponseWriter to be a http.Flusher")
	}

	vars := mux.Vars(r)
	if vars == nil {
		logger.Println("INVALID PATH")
		return
	}

	chunkSizeRaw := r.FormValue("chunk-size")
	chunkSize := int64(10 * 1024)
	if chunkSizeRaw != "" {
		chunkSize, err := strconv.ParseInt(chunkSizeRaw, 10, 0)
		if err != nil {
			logger.Printf("INVALID chunk size %s: %s", r.FormValue("chunk-size"), err)
		} else if chunkSize > 10*1024 {
			chunkSize = 10 * 1024
		}
	}

	n, err := strconv.ParseInt(vars["n"], 10, 64)
	if err != nil {
		logger.Println("INVALID path component", vars["n"], err)
		n = 100 * chunkSize
	} else if n > 100*chunkSize {
		n = 100 * chunkSize
	}

	filename := r.FormValue("filename")
	if filename == "" {
		filename = "data"
	}

	randGenerator := rand.New(rand.NewSource(1))
	seedRaw := r.FormValue("seed")
	if seedRaw != "" {
		seed, err := strconv.ParseInt(seedRaw, 10, 64)
		if err == nil {
			randGenerator = rand.New(rand.NewSource(seed))
		} else {
			logger.Printf("INVALID SEED %s %s", r.Form.Get("seed"), err)
		}
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	for n > 0 {
		writtedBytes := chunkSize
		if n < chunkSize {
			writtedBytes = n
		}

		io.CopyN(w, randGenerator, writtedBytes)
		flusher.Flush()
		n -= writtedBytes
		logger.Println("Write datas", writtedBytes)
	}
}

func ImgHandler(w http.ResponseWriter, r *http.Request) {
	acceptHeader := r.Header.Get("accept")

	if acceptHeader == "" {
		ImgPngHandler(w, r)
		return
	}

	if strings.Contains(acceptHeader, "image/webp") {
		ImgWebpHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/svg+xml") {
		ImgSVGHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/jpeg") {
		ImgJPEGHandler(w, r)
		return
	} else if strings.Contains(acceptHeader, "image/png") || strings.Contains(acceptHeader, "image/*") {
		ImgPngHandler(w, r)
		return
	} else {
		http.Error(w, "Invalid Accept", http.StatusNotAcceptable)
		return
	}

}

func ImgPngHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "pig_icon.png"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

func ImgJPEGHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "jackal.jpg"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

func ImgWebpHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "wolf_1.webp"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Write(data)
}

func ImgSVGHandler(w http.ResponseWriter, r *http.Request) {
	data, err := Resource(filepath.Join("images", "svg_logo.svg"))
	if err != nil {
		logger.InternalErrorPrint(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func BasicAuthHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars == nil {
		logger.Fatalln("INVALID path")
		return
	}
	user := vars["user"]
	passwd := vars["passwd"]

	if !checkBasicAuth(r, user, passwd) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Fake Realm"`)
		http.Error(w, "Incorrect User or Password", http.StatusUnauthorized)
		return
	}

	js, err := json.Marshal(map[string]interface{}{
		"authenticated": true,
		"user":          user,
	})
	if err != nil {
		logger.Fatalln(err)
	}

	fmt.Fprint(w, string(js))
}

/*
 * ====================================
 * WebApp Init
 * ====================================
 */

func GetMux() *mux.Router {
	var router = mux.NewRouter()
	var normalRouter = router.NewRoute().Subrouter()

	// 注册中间件
	registerMiddleware(normalRouter)

	// 注册API接口
	normalRouter.HandleFunc("/legacy", IndexHandler)
	normalRouter.HandleFunc("/ip", IPHandler)
	normalRouter.HandleFunc("/uuid", UUIDHandler)
	normalRouter.HandleFunc("/user-agent", UserAgentHandler)
	normalRouter.HandleFunc("/headers", HeadersHandler)
	normalRouter.HandleFunc("/get", GetHandler)
	normalRouter.HandleFunc("/image", ImgHandler)
	normalRouter.HandleFunc("/image/png", ImgPngHandler)
	normalRouter.HandleFunc("/image/jpeg", ImgJPEGHandler)
	normalRouter.HandleFunc("/image/webp", ImgWebpHandler)
	normalRouter.HandleFunc("/image/svg", ImgSVGHandler)
	normalRouter.HandleFunc("/base64/{value}", Base64Handler)
	normalRouter.HandleFunc("/bytes/{n}", BytesHandler)
	normalRouter.HandleFunc("/stream-bytes/{n}", StreamBytesHandler)
	normalRouter.HandleFunc("/basic-auth/{user}/{passwd}", BasicAuthHandler)

	// 注册静态文件
	router.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))))
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/swaggerui/dist")))

	return router
}

func registerMiddleware(router *mux.Router) {
	// TODO: 实现 JWT 认证
	//awm := middlewares.NewAuthMiddleware()
	//router.Use(awm.Middleware)
	router.Use(middlewares.JSONMiddleware)
}

func main() {
	var router = GetMux()
	handler := handlers.LoggingHandler(os.Stdout, router)

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

	c := make(chan os.Signal)
	go func() {
		logger.Println("Start server on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				logger.Println("Graceful shutdown the server")
				c <- os.Interrupt
			} else {
				logger.Fatalln(err)
			}
		}
	}()

	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)
	<-c

	os.Exit(0)
}
