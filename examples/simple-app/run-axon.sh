#!/bin/bash

# Script to build and run axon from the complete-app directory
# Usage: ./run-axon.sh [axon-arguments...]

set -e  # Exit on any error

echo "Building axon..."
cd ../../
go build -o axon ./cmd/axon
echo "✓ Axon built successfully"

echo "Running axon in simple-app..."
cd examples/simple-app
../../axon "$@"

go build -o simple-app ./main.go
echo "✓ simple-app built successfully"
