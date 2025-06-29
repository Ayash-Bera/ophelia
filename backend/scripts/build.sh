#!/bin/bash

echo "Building Arch Search Backend..."

# Clean previous builds
rm -rf dist/

# Create dist directory
mkdir -p dist/

# Build for current platform
echo "Building for current platform..."
go build -o dist/arch-search-server cmd/server/main.go

# Build for Linux (production)
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o dist/arch-search-server-linux cmd/server/main.go

# Build seeder
echo "Building seeder..."
go build -o dist/arch-search-seeder cmd/seed/main.go

echo "Build complete!"
echo "Files in dist/:"
ls -la dist/