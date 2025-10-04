#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting LibreCash Services...${NC}"
echo "================================"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi

echo -e "${YELLOW}🐳 Starting Docker services...${NC}"
docker compose up -d

# Wait for services to be ready
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"

# Wait for PostgreSQL (main database)
echo -e "${YELLOW}  📊 Waiting for main database...${NC}"
for i in {1..30}; do
    if docker compose exec -T -e PGPASSWORD=librecash db psql -h localhost -U librecash -d librecash -c "SELECT 1;" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Main database is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}  ❌ Main database failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Wait for PostgreSQL (test database)
echo -e "${YELLOW}  🧪 Waiting for test database...${NC}"
for i in {1..30}; do
    if docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test -c "SELECT 1;" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ Test database is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}  ❌ Test database failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Wait for RabbitMQ
echo -e "${YELLOW}  🐰 Waiting for RabbitMQ...${NC}"
for i in {1..30}; do
    if docker compose exec -T rabbit rabbitmqctl status >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ RabbitMQ is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}  ❌ RabbitMQ failed to start${NC}"
        exit 1
    fi
    sleep 1
done

echo ""
echo -e "${GREEN}🎉 All services are up and running!${NC}"
echo ""
echo -e "${BLUE}📊 Service Status:${NC}"
echo -e "  Main Database:  ${GREEN}postgresql://localhost:5432/librecash${NC}"
echo -e "  Test Database:  ${GREEN}postgresql://localhost:15433/librecash_test${NC}"
echo -e "  RabbitMQ:       ${GREEN}amqp://localhost:5672${NC}"
echo -e "  RabbitMQ UI:    ${GREEN}http://localhost:15672${NC} (guest/guest)"
echo ""
echo -e "${YELLOW}💡 Next steps:${NC}"
echo -e "  • Run ${BLUE}./initdb.sh${NC} to initialize database schema"
echo -e "  • Run ${BLUE}./run.sh${NC} to start the LibreCash bot"
echo -e "  • Run ${BLUE}./test.sh${NC} to run tests"
echo "================================"
