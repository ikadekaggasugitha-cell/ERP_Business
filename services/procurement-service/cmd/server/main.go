// procurement-service entrypoint: PostgreSQL RLS pool, REST API, outbox relay, and mdcache consumer.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/mdcache"
	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw@localhost:5432/procurement_db?sslmode=disable")
	addr := httpx.Getenv("HTTP_ADDR", ":8080")
	brokers := httpx.SplitCSV(os.Getenv("KAFKA_BROKERS"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()
	repo := repository.NewPurchaseOrderRepository(db)

	var wg sync.WaitGroup
	var pub *eventbus.Publisher
	var mdc *mdcache.Consumer
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "procurement-service", 2*time.Second, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, pub, 100)
			})
		}()
		log.Printf("outbox relay -> kafka %v", brokers)

		mdc = mdcache.New(repo, brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("procurement mdcache consumer <- kafka %v topics=%v", brokers, mdcache.Topics)
			if err := mdc.Run(ctx); err != nil {
				log.Printf("mdcache consumer stopped: %v", err)
			}
		}()
	}

	metrics := httpx.NewMetrics("procurement-service")
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
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("procurement-service",
		db.Ping,
		func(c context.Context) error { return eventbus.CheckBrokers(c, brokers) },
	))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	httpx.ServeHTTP(ctx, cancel, "procurement-service", addr, metrics.Wrap(httpx.Middleware(mux)), &wg, func() {
		if pub != nil {
			_ = pub.Close()
		}
		if mdc != nil {
			_ = mdc.Close()
		}
	})
}
