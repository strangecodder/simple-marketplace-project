package handler

import (
	"context"
	"fmt"
)

func HandleMail(ctx context.Context, body []byte) error {
	fmt.Println(string(body))
	return nil
}
