package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/seansa/lunar-backend-challenge/internal/config"
	"github.com/seansa/lunar-backend-challenge/internal/consumer"
	"github.com/seansa/lunar-backend-challenge/internal/service"
	eventstore "github.com/seansa/lunar-backend-challenge/internal/store/event/mysql"
	rocketstore "github.com/seansa/lunar-backend-challenge/internal/store/rocket/mysql"
)

const shutdownTimeout = 15 * time.Second

type dependencies struct {
	service  *service.Service
	consumer *consumer.Pool
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

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           newRouter(deps),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return serve(server, deps.consumer, stopConsumer)
}

// serve runs the API until the process is asked to stop. It then refuses new
// messages, lets the consumer pool drain, and stops the HTTP server.
func serve(server *http.Server, pool *consumer.Pool, stopConsumer context.CancelFunc) error {
	signals, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("api listening", "address", server.Addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		stopConsumer()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to run server: %w", err)
		}
		return nil
	case <-signals.Done():
	}

	slog.Info("shutdown requested, draining")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Closing the pool makes the in-flight requests fail with a retryable status
	// code, so the sender redelivers and nothing is lost.
	if err := pool.Shutdown(shutdownCtx); err != nil {
		slog.Warn("consumer pool did not drain in time", "error", err)
	}
	stopConsumer()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shut down server: %w", err)
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
		service:  service,
		consumer: pool,
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
