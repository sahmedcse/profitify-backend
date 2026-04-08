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
    ./cmd/lambda-fetch-tickers

# Runtime stage: AWS Lambda base image bundles the Runtime Interface
# Emulator, so the container can be invoked locally over HTTP. The
# default ENTRYPOINT (/lambda-entrypoint.sh) auto-detects local mode
# and launches the RIE which then runs /var/runtime/bootstrap.
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=builder /bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap
# For the provided runtime the handler arg is a placeholder; the
# bootstrap binary at /var/runtime/bootstrap is what actually runs.
CMD ["bootstrap"]
