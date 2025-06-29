#!/bin/bash

echo "Starting Arch Search development environment..."

# Start infrastructure services
cd docker
docker-compose up -d

echo "Waiting for services to be ready..."
sleep 10

# Check service health
echo "Checking service health..."
docker-compose ps

echo "Development environment ready!"
echo ""
echo "Services:"
echo "- PostgreSQL: localhost:5432"
echo "- Redis: localhost:6379" 
echo "- NATS: localhost:4222"
echo "- NATS Monitoring: http://localhost:8222"
echo ""
echo "Next steps:"
echo "1. cd ../.. && go run cmd/server/main.go"
echo "2. Test: curl http://localhost:8080/health"
echo ""
echo "To stop: docker-compose down"