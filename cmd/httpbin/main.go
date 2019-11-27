package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bwangelme/go-httpbin"
	"github.com/gorilla/handlers"
)

func main() {
	var router = httpbin.GetMux()
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
		log.Println("Start server on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Println("Graceful shutdown the server")
				c <- os.Interrupt
			} else {
				log.Fatalln(err)
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
