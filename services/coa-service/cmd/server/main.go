package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/consumer"
	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw@localhost:5432/coa_db?sslmode=disable")
	addr := httpx.Getenv("HTTP_ADDR", ":8080")
	brokers := httpx.SplitCSV(os.Getenv("KAFKA_BROKERS"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	repo := repository.NewAccountRepository(db)

	var wg sync.WaitGroup
	var pub *eventbus.Publisher
	var cons *consumer.Consumer
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		cons = consumer.New(repo, brokers)
		wg.Add(2)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "coa-service", 300*time.Millisecond, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, 50, pub)
			})
		}()
		go func() {
			defer wg.Done()
			if err := cons.Run(ctx); err != nil {
				log.Printf("[coa-consumer] stopped: %v", err)
			}
		}()
	}

	metrics := httpx.NewMetrics("coa-service")
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
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("coa-service", db.Ping, func(c context.Context) error {
		return eventbus.CheckBrokers(c, brokers)
	}))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	handler := metrics.Wrap(httpx.Middleware(mux))
	httpx.ServeHTTP(ctx, cancel, "coa-service", addr, handler, &wg, func() {
		if cons != nil {
			_ = cons.Close()
		}
		if pub != nil {
			_ = pub.Close()
		}
		db.Close()
	})
}
