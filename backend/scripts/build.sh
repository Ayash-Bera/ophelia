#!/bin/bash

echo "Building Arch Search Backend..."

# Clean previous builds
rm -rf dist/

# Create dist directory
mkdir -p dist/

# Build for current platform
echo "Building server for current platform..."
go build -o dist/arch-search-server cmd/server/main.go

echo "Building seeder for current platform..."
go build -o dist/arch-search-seeder cmd/seed/main.go

# Build for Linux (production)
echo "Building server for Linux..."
GOOS=linux GOARCH=amd64 go build -o dist/arch-search-server-linux cmd/server/main.go

echo "Building seeder for Linux..."
GOOS=linux GOARCH=amd64 go build -o dist/arch-search-seeder-linux cmd/seed/main.go

echo "Build complete!"
echo "Files in dist/:"
ls -la dist/

echo ""
echo "Usage:"
echo "  ./dist/arch-search-server                    # Start the API server"
echo "  ./dist/arch-search-seeder --help             # See seeder options"
echo "  ./dist/arch-search-seeder --dry-run          # Test without uploading"
echo "  ./dist/arch-search-seeder --limit 3          # Process only 3 pages"