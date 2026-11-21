package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/consumer"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	piiKey, err := platform.LoadPIIKey()
	if err != nil {
		log.Fatalf("payroll-service pii key init: %v", err)
	}
	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw:erp_dev_pw@localhost:5432/payroll_db")
	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("payroll-service db init: %v", err)
	}
	defer db.Close()

	repo := repository.NewPayrollRunRepository(db, piiKey)
	brokers := httpx.SplitCSV(httpx.Getenv("KAFKA_BROKERS", "localhost:9092"))
	var pub *eventbus.Publisher
	var cons *consumer.Consumer
	var wg sync.WaitGroup
	if len(brokers) > 0 {
		pub = eventbus.NewPublisher(brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventbus.RunOutboxRelay(ctx, "payroll-service", 300*time.Millisecond, func(c context.Context) (int, error) {
				return repo.DrainAndPublish(c, 50, pub)
			})
		}()
		cons = consumer.New(repo, brokers)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[payroll-service] consumer stopped: %v", err)
			}
		}()
	}

	metrics := httpx.NewMetrics("payroll-service")
	router := httpapi.NewRouter(
		repo,
		metrics,
		func(c context.Context) error { return db.Ping(c) },
		func(c context.Context) error { return eventbus.CheckBrokers(c, brokers) },
	)

	addr := httpx.Getenv("HTTP_ADDR", ":8098")
	httpx.ServeHTTP(ctx, cancel, "payroll-service", addr, router, &wg, func() {
		if cons != nil {
			_ = cons.Close()
		}
		if pub != nil {
			_ = pub.Close()
		}
	})
}
