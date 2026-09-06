package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"notification-service/internal/handler"
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
	var cfg config.Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatal(err)
	}

	rabbitClient := rabbit.NewRabbitClient()

	ctx := context.Background()

	err := configureRabbit(ctx, rabbitClient)

	if err != nil {
		log.Fatal(err)
	}

	// todo: переделать под конфигу
	err = rabbitClient.Connect("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port)))
}

func configureRabbit(ctx context.Context, client *rabbit.RabbitClient) error {
	rabbitErr := client.DeclareQueue("mail-queue")

	if rabbitErr != nil {
		return rabbitErr
	}

	rabbitErr = client.Consume(ctx, "mail-queue", handler.HandleMail)
	if rabbitErr != nil {
		return rabbitErr
	}
	return nil
}
