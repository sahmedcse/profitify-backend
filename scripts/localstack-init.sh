#!/bin/bash
set -euo pipefail

echo "Creating SQS queue..."
awslocal sqs create-queue --queue-name profitify-tickers

echo "Creating Step Functions state machine..."
awslocal stepfunctions create-state-machine \
  --name profitify-pipeline \
  --definition '{"StartAt":"Pass","States":{"Pass":{"Type":"Pass","End":true}}}' \
  --role-arn arn:aws:iam::000000000000:role/dummy

echo "LocalStack init complete."
