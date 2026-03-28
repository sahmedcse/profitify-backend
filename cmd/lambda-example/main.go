package main

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"

	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
)

// Event is the input payload for the Lambda function.
type Event struct {
	Name string `json:"name"`
}

// Response is the output payload for the Lambda function.
type Response struct {
	Message string `json:"message"`
}

func handleRequest(ctx context.Context, event Event) (Response, error) {
	logger := lambdautil.InitLogger()
	logger.Info("processing event", slog.String("name", event.Name))

	return Response{
		Message: "Hello, " + event.Name + "!",
	}, nil
}

func main() {
	lambda.Start(handleRequest)
}
