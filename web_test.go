package httpbin_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwangelme/go-httpbin"
	"github.com/gorilla/mux"
)

func TestIPHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ip", nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.RemoteAddr = "127.0.0.1"

	record := httptest.NewRecorder()
	handler := http.HandlerFunc(httpbin.IPHandler)
	handler.ServeHTTP(record, req)

	if status := record.Code; status != http.StatusOK {
		log.Fatalf("Error code %v, excepted %v\n", http.StatusOK, status)
	}

	expectedBody := `{"IP":"127.0.0.1"}`
	actualBody := record.Body.String()
	if expectedBody != actualBody {
		log.Fatalf("Unexcepted body %s, excepted body %s\n", actualBody, expectedBody)
	}
}

func TestBase64Handler(t *testing.T) {
	originTest := "hello world\n"
	base64EncodedVal := "aGVsbG8gd29ybGQK"
	path := fmt.Sprintf("/base64/%s", base64EncodedVal)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		log.Fatalln(err)
	}

	record := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/base64/{value}", httpbin.Base64Handler)
	router.ServeHTTP(record, req)

	if status := record.Code; status != http.StatusOK {
		log.Fatalf("Error code %v, excepted %v\n", http.StatusOK, status)
	}

	actualBody := record.Body.String()
	if originTest != actualBody {
		log.Fatalf("Unexcepted body %s, excepted body %s\n", actualBody, originTest)
	}
}

func TestImgHandler(t *testing.T) {
	// TODO: 测试 /image 接口，判断返回的图片类型
}
