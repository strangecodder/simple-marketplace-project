package server

import (
	"context"
	"database/sql"
	"fmt"
	listingv1 "gen/listing/v1"
	"listing-service/internal/config"
	"listing-service/internal/handler"
	"listing-service/internal/repository"
	"log"
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
	var cfg config.Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatal(err)
	}
	db, dbErr := connectDatabase(&cfg.Database)
	if dbErr != nil {
		log.Fatal(dbErr)
	}
	var repo repository.Repository
	repo = repository.NewRepository(db)

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	listingv1.RegisterListingServiceServer(grpcServer, handler.NewHandler(repo))
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatalf("failed to serve: %v", err)
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

	return db, nil
}
