// inventory-service entrypoint: PostgreSQL RLS pool, REST API, outbox relay, Saga/GR/mdcache consumers.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/consumer"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/mdcache"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw@localhost:5432/inventory_db?sslmode=disable")
	addr := httpx.Getenv("HTTP_ADDR", ":8080")
	brokers := httpx.SplitCSV(os.Getenv("KAFKA_BROKERS"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()
	repo := repository.NewStockRepository(db)

	var wg sync.WaitGroup
	var pub *eventbus.Publisher
	var cons *consumer.Consumer
	var mdc *mdcache.Consumer
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "inventory-service", 2*time.Second, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, pub, 100)
			})
		}()
		log.Printf("outbox relay -> kafka %v", brokers)

		cons = consumer.New(repo, brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("inventory consumer <- kafka %v topics=%v", brokers, consumer.Topics)
			if err := cons.Run(ctx); err != nil {
				log.Printf("consumer stopped: %v", err)
			}
		}()

		mdc = mdcache.New(repo, brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("inventory mdcache consumer <- kafka %v topics=%v", brokers, mdcache.Topics)
			if err := mdc.Run(ctx); err != nil {
				log.Printf("mdcache consumer stopped: %v", err)
			}
		}()
	}

	metrics := httpx.NewMetrics("inventory-service")
	mux := http.NewServeMux()
	httpapi.NewServer(repo).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		c, cc := context.WithTimeout(r.Context(), 2*time.Second)
		defer cc()
		if err := db.Ping(c); err != nil {
			http.Error(w, "db down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("inventory-service",
		db.Ping,
		func(c context.Context) error { return eventbus.CheckBrokers(c, brokers) },
	))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	httpx.ServeHTTP(ctx, cancel, "inventory-service", addr, metrics.Wrap(httpx.Middleware(mux)), &wg, func() {
		if pub != nil {
			_ = pub.Close()
		}
		if cons != nil {
			_ = cons.Close()
		}
		if mdc != nil {
			_ = mdc.Close()
		}
	})
}
