package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/consumer"
	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/httpapi"
	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/mdcache"
	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsn := httpx.Getenv("DATABASE_URL", "postgres://erp_app_rw:erp_dev_pw@localhost:5432/sales_db")
	db, err := platform.New(ctx, dsn)
	if err != nil {
		log.Fatalf("sales-service db init: %v", err)
	}
	defer db.Close()

	repo := repository.NewSalesOrderRepository(db)
	brokers := httpx.SplitCSV(httpx.Getenv("KAFKA_BROKERS", "localhost:9092"))
	pub := eventbus.NewPublisher(brokers)

	compCons := consumer.New(repo)
	compReader := eventbus.NewConsumer(brokers, "sales-service-compensation-v1", compCons.Topics())
	mdCons := mdcache.New(repo)
	mdReader := eventbus.NewConsumer(brokers, "sales-service-mdcache-v1", mdCons.Topics())

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		eventbus.RunOutboxRelay(ctx, "sales-service", 300*time.Millisecond, func(c context.Context) (int, error) {
			return repo.DrainAndPublish(c, 50, pub)
		})
	}()
	go func() {
		defer wg.Done()
		if err := compReader.Run(ctx, compCons.Handle); err != nil {
			log.Printf("[sales-consumer] stopped: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := mdReader.Run(ctx, mdCons.Handle); err != nil {
			log.Printf("[sales-mdcache] stopped: %v", err)
		}
	}()

	metrics := httpx.NewMetrics("sales-service")
	router := httpapi.NewRouter(
		repo,
		metrics,
		func(c context.Context) error { return db.Ping(c) },
		func(c context.Context) error { return eventbus.CheckBrokers(c, brokers) },
	)

	addr := httpx.Getenv("HTTP_ADDR", ":8092")
	httpx.ServeHTTP(ctx, cancel, "sales-service", addr, router, &wg, func() {
		_ = compReader.Close()
		_ = mdReader.Close()
		_ = pub.Close()
	})
}
