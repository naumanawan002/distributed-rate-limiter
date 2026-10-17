package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nauman/distributed-rate-limiter/pkg/gateway"
	"github.com/nauman/distributed-rate-limiter/pkg/limiter"
)

func main() {
	listenAddr := flag.String("addr", ":8080", "Gateway listening address")
	upstreamAddr := flag.String("upstream", "http://127.0.0.1:8081", "Upstream backend URL")
	capacity := flag.Int64("capacity", 50, "Token bucket burst capacity")
	rate := flag.Float64("rate", 20.0, "Token bucket refill rate per second")
	windowLimit := flag.Int64("window-limit", 100, "Window limit for sliding/fixed window")
	windowDuration := flag.Duration("window", time.Minute, "Window duration")
	flag.Parse()

	targetURL, err := url.Parse(*upstreamAddr)
	if err != nil {
		log.Fatalf("Invalid upstream URL: %v", err)
	}

	// Initialize algorithms
	tb := limiter.NewTokenBucket(*capacity, *rate)
	sw := limiter.NewSlidingWindowCounter(*windowLimit, *windowDuration)
	fw := limiter.NewFixedWindow(*windowLimit, *windowDuration)

	gw := gateway.NewGateway(targetURL, tb, sw, fw)

	srv := &http.Server{
		Addr:         *listenAddr,
		Handler:      gw,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("==========================================================")
		log.Printf(" API Gateway with Distributed Rate Limiting running on %s\n", *listenAddr)
		log.Printf(" Upstream Target: %s\n", targetURL)
		log.Printf(" Default Token Bucket: Capacity=%d, RefillRate=%.1f/s\n", *capacity, *rate)
		log.Printf(" Sliding/Fixed Window: Limit=%d per %v\n", *windowLimit, *windowDuration)
		log.Println(" Switch algorithms per-request via: 'X-RateLimit-Algorithm: sliding_window|fixed_window|token_bucket'")
		log.Println("==========================================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failure: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Initiating graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("Gateway stopped cleanly.")
}
