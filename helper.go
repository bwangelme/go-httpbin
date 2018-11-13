package main

/*
 * Helper Func
 */

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func checkBasicAuth(r *http.Request, user string, passwd string) bool{
	User, Passwd, ok := r.BasicAuth()
	fmt.Println(User, Passwd, ok)
	if !ok {
		return false
	}

	return (User == user && Passwd == passwd)
}

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

func Resource(filename string) (data []byte, err error) {
	if !filepath.IsAbs(filename) {
		filename = filepath.Join("static", filename)
	}

	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)
	for {
		n, err := fd.Read(buf)
		if n > 0 {
			// 注意buf的长度是1024，尾部可能有无效数据。故使用buf[:n]
			data = append(data, buf[:n]...)
		} else {
			if err == io.EOF && n == 0 {
				return data, nil
			} else {
				return nil, err
			}
		}
	}
}
