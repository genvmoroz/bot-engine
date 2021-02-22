package router

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
)

type Router struct {
	srv      *http.Server
	memStats runtime.MemStats
}

func New(port int32) *Router {
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port)}

	return &Router{
		srv: srv,
	}
}

func (r *Router) ListenAndServeWithContext(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	http.HandleFunc("/debug/info", r.info)

	go func() {
		if err := r.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to listen and serve: %s", err.Error())
		}
	}()

	<-ctx.Done()
	if err := r.srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	return nil
}

var infoPattern = "" +
	"Runtime OS: %s \n" +
	"Runtime ARCH: %s \n" +
	"Goroutines count: %d; \n" +
	"Allocated heap objects: %0.3f Mb \n" +
	"Total allocated memory for heap objects for the life of the program: %0.3f Mb \n" +
	"Total memory obtained from the OS: %0.3f Mb \n" +
	"The number of completed GC cycles: %d \n"

func (r *Router) info(w http.ResponseWriter, _ *http.Request) {
	runtime.ReadMemStats(&r.memStats)

	info := fmt.Sprintf(
		infoPattern,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumGoroutine(),
		bToMb(r.memStats.HeapAlloc),
		bToMb(r.memStats.TotalAlloc),
		bToMb(r.memStats.Sys),
		r.memStats.NumGC,
	)

	_, err := io.WriteString(w, info)
	if err != nil {
		log.Printf("failed to write the string, path: /info, err: %s", err.Error())
	}
}

func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}
