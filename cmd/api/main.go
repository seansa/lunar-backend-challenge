package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/seansa/lunar-backend-challenge/internal/config"
	"github.com/seansa/lunar-backend-challenge/internal/consumer"
	"github.com/seansa/lunar-backend-challenge/internal/service"
	eventstore "github.com/seansa/lunar-backend-challenge/internal/store/event/mysql"
	rocketstore "github.com/seansa/lunar-backend-challenge/internal/store/rocket/mysql"
)

type dependencies struct {
	service          *service.Service
	eventRepository  *eventstore.Repository
	rocketRepository *rocketstore.Repository
	consumer         *consumer.Pool
}

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	db, err := openDatabase(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	deps, err := makeDependencies(db, cfg)
	if err != nil {
		return fmt.Errorf("make dependencies: %w", err)
	}

	consumerCtx, stopConsumer := context.WithCancel(context.Background())
	defer stopConsumer()
	deps.consumer.Start(consumerCtx)

	r := newRouter(deps)

	if err := r.Run(fmt.Sprintf(":%d", cfg.HTTPPort)); err != nil {
		return fmt.Errorf("failed to run server: %v", err)
	}
	return nil
}

func makeDependencies(db *sql.DB, cfg config.Config) (*dependencies, error) {
	eventRepository := eventstore.New(db)
	rocketRepository := rocketstore.New(db)
	service := service.New(eventRepository, rocketRepository)

	consumerService := consumer.New(eventRepository, service)

	pool := consumer.NewPool(consumerService, consumer.PoolOptions{
		Workers:        cfg.ConsumerWorkers,
		QueueSize:      cfg.ConsumerQueueSize,
		ProcessTimeout: cfg.ConsumerProcessTimeout,
	})

	return &dependencies{
		service:          service,
		eventRepository:  eventRepository,
		rocketRepository: rocketRepository,
		consumer:         pool,
	}, nil
}

func openDatabase(ctx context.Context, cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	// The database may still be starting when the container comes up.
	var lastErr error
	for attempt := 1; attempt <= cfg.DBConnectRetries; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		lastErr = db.PingContext(pingCtx)
		cancel()

		if lastErr == nil {
			break
		}
		slog.Warn("database not ready yet", "attempt", attempt, "error", lastErr)

		select {
		case <-ctx.Done():
			_ = db.Close()
			return nil, ctx.Err()
		case <-time.After(cfg.DBConnectRetryDelay):
		}
	}
	if lastErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to database: %w", lastErr)
	}

	return db, nil
}
