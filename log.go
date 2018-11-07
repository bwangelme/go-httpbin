package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

type WebLogger struct {
	*log.Logger
	req    *http.Request
}

func NewWebLogger() *WebLogger {
	basicLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	return &WebLogger{
		Logger: basicLogger,
	}
}

func (l *WebLogger) InternalErrorPrint(w http.ResponseWriter, v ...interface{}) {
	msg := fmt.Sprint(v...)
	http.Error(w, msg, http.StatusInternalServerError)

	l.Logger.Output(2, msg)
}

func (l *WebLogger) InternalErrorPrintf(w http.ResponseWriter, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	http.Error(w, msg, http.StatusInternalServerError)

	l.Logger.Output(2, msg)
}
