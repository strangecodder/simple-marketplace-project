package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"payment-service/internal/handler"
	"payment-service/internal/repository"
	paymentv1 "simple-marketplace-project/gen/payment/v1"
	"simple-marketplace-project/pkg/config"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
)

type Server struct {
	paymentService paymentv1.PaymentServiceServer
}

func LaunchServer() {
	var cfg config.Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatal(err)
	}
	db, dbError := connectDatabase(&cfg.Database)
	if dbError != nil {
		log.Fatal(dbError)
	}

	var paymentRepository repository.PaymentRepository
	paymentRepository = repository.NewPaymentRepository(db)
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, handler.NewPaymentHandler(paymentRepository))
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatal(err)
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
