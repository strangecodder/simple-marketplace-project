package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	listingv1 "simple-marketplace-project/gen/listing/v1"
	"simple-marketplace-project/pkg/config"

	"listing-service/internal/handler"
	"listing-service/internal/repository"
	"net"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	listingv1.ListingServiceServer
}

func LaunchServer() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var cfg config.Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	db, dbErr := connectDatabase(&cfg.Database)
	if dbErr != nil {
		logger.Error(dbErr.Error())
		panic(dbErr)
	}
	var repo repository.Repository
	repo = repository.NewRepository(db, logger)

	listingv1.RegisterListingServiceServer(grpcServer, handler.NewHandler(repo, logger))
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	if err := grpcServer.Serve(listen); err != nil {
		logger.Error("failed to serve: %s", err.Error())
		panic(err)
	}
}

func connectDatabase(cfg *config.DBConfig) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	migrationErr := runMigrations(db, cfg.MigrationPath)
	if migrationErr != nil {
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB, sourceUrl string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		sourceUrl,
		"postgres", driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
