package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIPHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ip", nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.RemoteAddr = "127.0.0.1"

	record := httptest.NewRecorder()
	handler := http.HandlerFunc(IPHandler)
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
	router.HandleFunc("/base64/{value}", Base64Handler)
	router.ServeHTTP(record, req)

	if status := record.Code; status != http.StatusOK {
		log.Fatalf("Error code %v, excepted %v\n", http.StatusOK, status)
	}

	actualBody := record.Body.String()
	if originTest != actualBody {
		log.Fatalf("Unexcepted body %s, excepted body %s\n", actualBody, originTest)
	}
}
