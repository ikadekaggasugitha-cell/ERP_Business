package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	piiKey, err := platform.LoadPIIKey()
	if err != nil {
		log.Fatalf("hr-service pii key init: %v", err)
	}
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw:erp_dev_pw@localhost:5432/hr_db")
	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("hr-service db init: %v", err)
	}
	defer db.Close()

	repo := repository.NewEmployeeRepository(db, piiKey)
	brokers := httpx.SplitCSV(httpx.Getenv("KAFKA_BROKERS", "localhost:9092"))
	var pub *eventbus.Publisher
	var wg sync.WaitGroup
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "hr-service", 300*time.Millisecond, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, 50, pub)
			})
		}()
	}

	metrics := httpx.NewMetrics("hr-service")
	router := httpapi.NewRouter(
		repo,
		metrics,
		func(c context.Context) error { return db.Ping(c) },
		func(c context.Context) error { return eventbus.CheckBrokers(c, brokers) },
	)

	addr := httpx.Getenv("HTTP_ADDR", ":8097")
	httpx.ServeHTTP(ctx, cancel, "hr-service", addr, router, &wg, func() {
		if pub != nil {
			_ = pub.Close()
		}
	})
}
