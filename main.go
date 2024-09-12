package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func echo(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 32)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			_, err = c.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func initEchoServer(wg *sync.WaitGroup, port string) net.Listener {
	l, err := net.Listen("tcp", fmt.Sprint(":", port))
	if err != nil {
		fmt.Println("Init ECHO server failed:", err)
		os.Exit(1)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			c, err := l.Accept()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
					fmt.Println("ECHO server stopped.")
				} else {
					fmt.Println("ECHO server exited with an error:", err)
				}
				break
			}

			go echo(c)
		}
	}()
	return l
}

type HttpServer struct {
	graceful int64
	svr      *http.Server
}

func (svr *HttpServer) PrepareShutdown() {
	atomic.StoreInt64(&svr.graceful, 1)
}

func (svr *HttpServer) Shutdown() {
	svr.svr.Shutdown(context.TODO())
}

func initHTTPServer(wg *sync.WaitGroup, port string) *HttpServer {
	wrapper := &HttpServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt64(&wrapper.graceful) > 0 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte("ready"))
	})

	wrapper.svr = &http.Server{Addr: fmt.Sprint(":", port), Handler: mux}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("HTTP server started.")
		err := wrapper.svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Println("Server exited with error:", err)
			os.Exit(1)
		} else {
			fmt.Println("HTTP server stopped.")
		}
	}()

	return wrapper
}

func initSignalHandler(wg *sync.WaitGroup, httpServer *HttpServer, echoServer net.Listener) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		fmt.Println("Signal handler started.")
		c := make(chan os.Signal, 1)
		defer close(c)
		signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
		tmr := time.NewTimer(30 * time.Second)
		tmr.Stop()
		gracefulEntered := false
	LOOP:
		for {
			select {
			case s := <-c:
				fmt.Println("Received signal:", s)
				if !gracefulEntered {
					fmt.Println("Graceful period started.")
					tmr.Reset(30 * time.Second)
					echoServer.Close()
					httpServer.PrepareShutdown()
					gracefulEntered = true
				}
			case <-tmr.C:
				httpServer.Shutdown()
				fmt.Println("Graceful period ended.")
				break LOOP
			}
		}
		fmt.Println("Signal handler stopped.")
	}()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:", os.Args[0], "<port> <port1>")
		fmt.Println()
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	initSignalHandler(wg, initHTTPServer(wg, os.Args[1]), initEchoServer(wg, os.Args[2]))
	wg.Wait()
}
