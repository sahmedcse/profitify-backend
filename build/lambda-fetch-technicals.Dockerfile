# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags lambda.norpc \
    -o /bootstrap \
    ./cmd/lambda-fetch-technicals

FROM public.ecr.aws/lambda/provided:al2023
COPY --from=builder /bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap
CMD ["bootstrap"]
