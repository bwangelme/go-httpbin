package httpbin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bwangelme/go-httpbin/middlewares"
	"github.com/google/uuid"
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

func unescaped(x string) interface{} { return template.HTML(x) }

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
		"URL_CONFIG":       URL_CONFIG,
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
	// TODO 写一个功能函数，统一获取相应内容，类似于 py 中的 get_dict
	result := make(map[string]interface{})
	queryArgs, err := getQueryArgs(r)
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	result["origin"] = getPeerIP(r)
	result["url"] = fmt.Sprintf("%s://%s%s", getRequestScheme(r), r.Host, r.URL.RequestURI())
	result["headers"] = getHeadersMap(r.Header)
	result["args"] = queryArgs

	// TODO 中间件统一写返回值的代码
	js, err := json.Marshal(result)
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	fmt.Fprintf(w, string(js))
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]interface{})
	queryArgs, err := getQueryArgs(r)
	if err != nil {
		logger.InternalErrorPrint(w, err.Error())
		return
	}

	result["url"] = fmt.Sprintf("%s://%s%s", getRequestScheme(r), r.Host, r.URL.RequestURI())
	result["args"] = queryArgs
	result["origin"] = getPeerIP(r)
	result["headers"] = getHeadersMap(r.Header)

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

func redirectToHandler(w http.ResponseWriter, r *http.Request, url string, statusCode int) {
	w.Header().Set("Location", url)

	if !(statusCode >= 300 && statusCode < 400) {
		statusCode = http.StatusFound
	}

	w.WriteHeader(statusCode)

}

func RedirectToGetHandler(w http.ResponseWriter, r *http.Request) {
	var statusCode int
	url := mux.Vars(r)["url"]
	statusCodeStrs, ok := r.URL.Query()["status_code"]
	if !ok {
		statusCode = 0
	} else {
		// 由于 Atoi 出错时也是返回0，所以这里不用判断错误
		statusCode, _ = strconv.Atoi(statusCodeStrs[0])
	}

	redirectToHandler(w, r, url, statusCode)
	return
}

func RedirectToFormHandler(w http.ResponseWriter, r *http.Request) {
	var statusCode int

	r.ParseForm()
	data := r.Form
	urls, exist := data["url"]
	if !exist {
		GetHandler(w, r)
		return
	}
	url := urls[0]

	statusCodeStrs, ok := data["status_code"]
	if ok {
		statusCode, _ = strconv.Atoi(statusCodeStrs[0])
	}
	redirectToHandler(w, r, url, statusCode)
	return
}

/*
 * ====================================
 * WebApp Init
 * ====================================
 */

func GetMux() *mux.Router {
	var router = mux.NewRouter()
	var apiRouter = router.NewRoute().Subrouter()
	var imgRouter = router.NewRoute().Subrouter()

	// 注册中间件
	registerMiddleware(apiRouter)

	// 注册API接口
	apiRouter.HandleFunc("/legacy", IndexHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/uuid", UUIDHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/base64/{value}", Base64Handler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/bytes/{n}", BytesHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/stream-bytes/{n}", StreamBytesHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/basic-auth/{user}/{passwd}", BasicAuthHandler).Methods(http.MethodGet, http.MethodHead)

	//TODO HTTP Methods
	apiRouter.HandleFunc("/delete", DeleteHandler).Methods(http.MethodDelete)
	apiRouter.HandleFunc("/get", GetHandler).Methods(http.MethodGet, http.MethodHead)

	// Request Inspection
	apiRouter.HandleFunc("/headers", HeadersHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/ip", IPHandler).Methods(http.MethodGet, http.MethodHead)
	apiRouter.HandleFunc("/user-agent", UserAgentHandler).Methods(http.MethodGet, http.MethodHead)

	// Redirects
	apiRouter.HandleFunc("/redirect-to", RedirectToGetHandler).Methods(http.MethodGet, http.MethodHead).Queries("url", "{url:.+}")
	apiRouter.HandleFunc("/redirect-to", RedirectToFormHandler).Methods(http.MethodPut, http.MethodPatch, http.MethodPost)

	// Images
	imgRouter.HandleFunc("/image", ImgHandler).Methods(http.MethodGet, http.MethodHead)
	imgRouter.HandleFunc("/image/png", ImgPngHandler).Methods(http.MethodGet, http.MethodHead)
	imgRouter.HandleFunc("/image/jpeg", ImgJPEGHandler).Methods(http.MethodGet, http.MethodHead)
	imgRouter.HandleFunc("/image/webp", ImgWebpHandler).Methods(http.MethodGet, http.MethodHead)
	imgRouter.HandleFunc("/image/svg", ImgSVGHandler).Methods(http.MethodGet, http.MethodHead)
	imgRouter.HandleFunc("/image/gif", ImgGIFHandler).Methods(http.MethodGet, http.MethodHead)

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
