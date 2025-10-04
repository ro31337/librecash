#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🤖 Starting LibreCash Bot...${NC}"
echo "================================"

# Check if services are running
echo -e "${YELLOW}🔍 Checking services...${NC}"

# Check Docker services
if ! docker compose ps | grep -q "Up"; then
    echo -e "${RED}❌ Docker services are not running!${NC}"
    echo -e "${YELLOW}💡 Run ${BLUE}./up.sh${NC} first to start services${NC}"
    exit 1
fi

# Check main database
if ! docker compose exec -T -e PGPASSWORD=librecash db psql -h localhost -U librecash -d librecash -c "SELECT 1;" >/dev/null 2>&1; then
    echo -e "${RED}❌ Main database is not accessible!${NC}"
    echo -e "${YELLOW}💡 Run ${BLUE}./initdb.sh${NC} to initialize the database${NC}"
    exit 1
fi

# Check if database schema exists
if ! docker compose exec -T -e PGPASSWORD=librecash db psql -h localhost -U librecash -d librecash -c "SELECT 1 FROM users LIMIT 1;" >/dev/null 2>&1; then
    echo -e "${RED}❌ Database schema is not initialized!${NC}"
    echo -e "${YELLOW}💡 Run ${BLUE}./initdb.sh${NC} to initialize the database schema${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All services are ready${NC}"

# Stop any existing bot instance
if pgrep -f "librecash_bot" >/dev/null; then
    echo -e "${YELLOW}🔄 Stopping existing bot instance...${NC}"
    pkill -f librecash_bot
    sleep 2
fi

# Build the application
echo -e "${YELLOW}🔨 Building LibreCash bot...${NC}"
if ! go build -o librecash_bot; then
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Build successful${NC}"

# Start the bot
echo -e "${YELLOW}🚀 Starting LibreCash bot...${NC}"
echo ""
echo -e "${BLUE}📱 Bot Status:${NC}"
echo -e "  Bot Name: ${GREEN}@librecash_bot${NC}"
echo -e "  Log Level: ${GREEN}INFO${NC}"
echo ""
echo -e "${YELLOW}💡 Press Ctrl+C to stop the bot${NC}"
echo "================================"
echo ""

# Run the bot
./librecash_bot
