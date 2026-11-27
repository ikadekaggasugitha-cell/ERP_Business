package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/consumer"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw@localhost:5432/fixedasset_db?sslmode=disable")
	addr := httpx.Getenv("HTTP_ADDR", ":8080")
	brokers := httpx.SplitCSV(os.Getenv("KAFKA_BROKERS"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	repo := repository.NewFixedAssetRepository(db)

	var wg sync.WaitGroup
	var pub *eventbus.Publisher
	var cons *consumer.Consumer
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "fixedasset-service", 300*time.Millisecond, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, 50, pub)
			})
		}()
		cons = consumer.New(repo, brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("fixedasset consumer stopped: %v", err)
			}
		}()
	}

	metrics := httpx.NewMetrics("fixedasset-service")
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
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("fixedasset-service", db.Ping, func(c context.Context) error {
		return eventbus.CheckBrokers(c, brokers)
	}))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	httpx.ServeHTTP(ctx, cancel, "fixedasset-service", addr, metrics.Wrap(httpx.Middleware(mux)), &wg, func() {
		if cons != nil {
			_ = cons.Close()
		}
		if pub != nil {
			_ = pub.Close()
		}
		db.Close()
	})
}
