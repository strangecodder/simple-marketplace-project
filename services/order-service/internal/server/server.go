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

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
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

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	grpcServer := grpc.NewServer()

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	var rabbitProducer rabbit.RabbitProducer
	rabbitProducer, rabbitErr := createRabbitClient(&cfg.Rabbit, logger)
	if rabbitErr != nil {
		logger.Error(rabbitErr.Error())
		panic(rabbitErr)
	}

	db, dbError := connectDatabase(&cfg.Database, logger)
	if dbError != nil {
		logger.Error(dbError.Error())
		panic(dbError.Error())
	}

	var repo repository.OrderRepository
	repo = repository.NewOrderRepository(db, logger)

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	orderv1.RegisterOrderServiceServer(grpcServer, handler.NewHandler(repo, rabbitProducer, logger))
	if err := grpcServer.Serve(listen); err != nil {
		logger.Error(err.Error())
		panic(err)
	}
}

func connectDatabase(cfg *config.DBConfig, logger *slog.Logger) (*sql.DB, error) {
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

	// todo:
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
