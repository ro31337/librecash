#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛑 Stopping LibreCash Services...${NC}"
echo "================================"

# Stop LibreCash bot if running
echo -e "${YELLOW}🤖 Stopping LibreCash bot...${NC}"
if pgrep -f "librecash_bot" >/dev/null; then
    pkill -f librecash_bot
    echo -e "${GREEN}  ✅ LibreCash bot stopped${NC}"
else
    echo -e "${YELLOW}  ℹ️  LibreCash bot was not running${NC}"
fi

# Stop Docker services
echo -e "${YELLOW}🐳 Stopping Docker services...${NC}"
docker compose down

# Optional: Remove volumes (uncomment if you want to clear all data)
# echo -e "${YELLOW}🗑️  Removing volumes...${NC}"
# docker compose down -v

echo ""
echo -e "${GREEN}🎉 All services stopped!${NC}"
echo ""
echo -e "${YELLOW}💡 To remove all data (databases, queues), run:${NC}"
echo -e "  ${BLUE}docker compose down -v${NC}"
echo "================================"
