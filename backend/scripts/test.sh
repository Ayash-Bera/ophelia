#!/bin/bash

echo "Running tests..."

# Run unit tests
go test -v ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html

echo "Tests complete!"
echo "Coverage report: coverage.html"