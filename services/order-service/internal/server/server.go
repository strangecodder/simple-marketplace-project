package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"order-service/internal/handler"
	"order-service/internal/repository"
	orderv1 "simple-marketplace-project/gen/order/v1"
	"simple-marketplace-project/pkg/config"
	"simple-marketplace-project/pkg/rabbit"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
)

type Server struct {
	orderv1.OrderServiceServer
}

func NewServer() *Server {
	return &Server{}
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

	var repo repository.OrderRepository
	repo = repository.NewOrderRepository(db)

	var rabbitProducer rabbit.RabbitProducer

	rabbitProducer, err := createRabbitClient(&cfg.Rabbit)
	if err != nil {
		log.Fatal(err)
	}

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, handler.NewHandler(repo, rabbitProducer))
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

func createRabbitClient(cfg *config.RabbitConfig) (*rabbit.RabbitClient, error) {
	rabbitClient := rabbit.NewRabbitClient()

	err := configureRabbit(context.Background(), rabbitClient)

	if err != nil {
		return nil, err
	}

	return rabbitClient, nil
}

func configureRabbit(ctx context.Context, client *rabbit.RabbitClient) error {
	rabbitErr := client.DeclareQueue("mail-queue")

	if rabbitErr != nil {
		return rabbitErr
	}

	//rabbitErr = client.Consume(ctx, "mail-queue", handler.HandleMail)
	if rabbitErr != nil {
		return rabbitErr
	}
	return nil
}
