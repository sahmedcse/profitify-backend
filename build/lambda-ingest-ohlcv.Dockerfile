# syntax=docker/dockerfile:1

# Build stage: compile the Lambda handler as a static linux binary.
FROM golang:1.23-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags lambda.norpc \
    -o /bootstrap \
    ./cmd/lambda-ingest-ohlcv

# Runtime stage: AWS Lambda base image bundles the Runtime Interface
# Emulator, so the container can be invoked locally over HTTP.
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=builder /bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap
CMD ["bootstrap"]
