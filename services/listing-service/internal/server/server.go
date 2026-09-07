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

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
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
	db, dbErr := connectDatabase(&cfg.Database, logger)
	if dbErr != nil {
		logger.Error(dbErr.Error())
		panic(dbErr)
	}
	var repo repository.Repository
	repo = repository.NewRepository(db)

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	grpcServer := grpc.NewServer()
	listingv1.RegisterListingServiceServer(grpcServer, handler.NewHandler(repo))
	if err := grpcServer.Serve(listen); err != nil {
		logger.Error("failed to serve: %s", err.Error())
		panic(err)
	}
}

func connectDatabase(cfg *config.DBConfig, logger *slog.Logger) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return db, nil
}
