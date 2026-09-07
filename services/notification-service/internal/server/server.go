package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"notification-service/internal/handler"
	"os"
	"simple-marketplace-project/pkg/config"
	"simple-marketplace-project/pkg/rabbit"

	"github.com/ilyakaznacheev/cleanenv"
)

type Server struct {
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

	rabbitClient := rabbit.NewRabbitClient(logger)

	ctx := context.Background()

	err := configureRabbit(ctx, rabbitClient, logger)

	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// todo: переделать под конфигу
	err = rabbitClient.Connect("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	log.Fatal(net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port)))
}

func configureRabbit(ctx context.Context, client *rabbit.RabbitClient, logger *slog.Logger) error {
	rabbitErr := client.DeclareQueue("mail-queue")

	if rabbitErr != nil {
		logger.Error(rabbitErr.Error())
		return rabbitErr
	}

	rabbitErr = client.Consume(ctx, "mail-queue", handler.HandleMail)
	if rabbitErr != nil {
		logger.Error(rabbitErr.Error())
		return rabbitErr
	}
	return nil
}
