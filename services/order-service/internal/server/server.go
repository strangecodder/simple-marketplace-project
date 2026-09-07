package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"order-service/internal/handler"
	"order-service/internal/repository"
	"os"
	orderv1 "simple-marketplace-project/gen/order/v1"
	"simple-marketplace-project/pkg/config"
	"simple-marketplace-project/pkg/rabbit"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	orderv1.OrderServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func LaunchServer() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	var cfg config.Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	db, dbError := connectDatabase(&cfg.Database, logger)
	if dbError != nil {
		logger.Error(dbError.Error())
		panic(dbError.Error())
	}

	var repo repository.OrderRepository
	repo = repository.NewOrderRepository(db, logger)

	var rabbitProducer rabbit.RabbitProducer

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	// изначально NOT_SERVING — сервер поднят, но Rabbit ещё не готов
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	rabbitProducer, rabbitErr := createRabbitClient(&cfg.Rabbit, logger)
	if rabbitErr != nil {
		logger.Error(rabbitErr.Error())
		panic(rabbitErr)
	}

	orderv1.RegisterOrderServiceServer(grpcServer, handler.NewHandler(repo, rabbitProducer, logger))
	if err := grpcServer.Serve(listen); err != nil {
		slog.Error(err.Error())
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

func createRabbitClient(cfg *config.RabbitConfig, logger *slog.Logger) (*rabbit.RabbitClient, error) {
	rabbitClient := rabbit.NewRabbitClient(logger)

	err := rabbitClient.DeclareQueue("mail-queue")
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	//err := configureRabbit(context.Background(), rabbitClient, logger)

	//rabbitErr = client.Consume(ctx, "mail-queue", handler.HandleMail)

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return rabbitClient, nil
}

func connectRabbitWithRetry(dsn string, logger *slog.Logger) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(dsn)
		if err == nil {
			return conn, nil
		}
		logger.Warn("rabbit not ready, retrying", "attempt", i+1, "err", err)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}
