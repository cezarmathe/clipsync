package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/cezarmathe/clipsync/internal"
)

const (
	PORT = 42424
)

var (
	mu    = new(sync.Mutex)
	peers = make([]string, 0)
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := internal.NewBasicStore()
	if err != nil {
		panic(err)
	}

	observer := internal.NewPollingObserver()

	sd, err := internal.NewServiceDiscovery(PORT, func(s []string) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Printf("updated peers: %v\n", s)
		peers = s
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("hello, this is clipsync")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := observer.Run(ctx); err != nil {
			panic(err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sd.Run(ctx); err != nil {
			panic(err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch := observer.GetChan()
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-ch:
				if !ok {
					return
				}
				if x, _ := store.Get(); x != v {
					store.Update(v)
					fmt.Println("update clipboard: local")
					v, ua := store.Get()
					go func() {
						mu.Lock()
						defer mu.Unlock()
						for _, peer := range peers {
							p := peer
							go func() {
								b := bytes.NewBufferString(v)
								http.Post(
									fmt.Sprintf("http://%s/?ua=%d", p, ua.Unix()),
									"text/plain",
									b,
								)
							}()
						}
					}()
				}
			}
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		ua, err := strconv.Atoi(q["ua"][0])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		updatedAt := time.Unix(int64(ua), 0)
		if _, oldUa := store.Get(); oldUa.After(updatedAt) {
			w.WriteHeader(http.StatusConflict)
			fmt.Println("received older clipboard, not updating")
			return
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := clipboard.WriteAll(string(b)); err != nil {
			panic(err)
		}
		store.Update(string(b))
		fmt.Println("update clipboard: remote")
		w.WriteHeader(http.StatusOK)
	})
	httpServer := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", strconv.Itoa(PORT)),
		Handler: mux,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stop()
			panic(err)
		}
	}()

	fmt.Println("running")

	<-ctx.Done()

	fmt.Println("stopping")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		panic(err)
	}
	wg.Wait()
	fmt.Println("bye bye")
}
