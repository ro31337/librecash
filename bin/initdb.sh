#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🗄️ Initializing LibreCash Database...${NC}"
echo "================================"

# Check if Docker services are running
if ! docker compose ps | grep -q "Up"; then
    echo -e "${RED}❌ Docker services are not running!${NC}"
    echo -e "${YELLOW}💡 Run ${BLUE}./up.sh${NC} first to start services${NC}"
    exit 1
fi

# Wait for database to be ready
echo -e "${YELLOW}⏳ Waiting for database to be ready...${NC}"
for i in {1..30}; do
    if docker compose exec -T -e PGPASSWORD=librecash db psql -h localhost -U librecash -d librecash -c "SELECT 1;" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Database is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ Database failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# Initialize database schema
echo -e "${YELLOW}🔄 Initializing database schema...${NC}"
echo -e "${YELLOW}  ⚠️  This will DROP and recreate all tables!${NC}"

# Check for -y flag for non-interactive mode
if [[ "$1" != "-y" ]]; then
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}❌ Database initialization cancelled${NC}"
        exit 0
    fi
fi

docker compose exec -T -e PGPASSWORD=librecash db psql -h localhost -U librecash -d librecash < db/init.sql

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database schema initialized successfully${NC}"
else
    echo -e "${RED}❌ Failed to initialize database schema${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 Database initialization complete!${NC}"
echo ""
echo -e "${BLUE}📊 Database Info:${NC}"
echo -e "  Host: ${GREEN}localhost:5432${NC}"
echo -e "  Database: ${GREEN}librecash${NC}"
echo -e "  User: ${GREEN}librecash${NC}"
echo ""
echo -e "${YELLOW}💡 You can now run ${BLUE}./run.sh${NC} to start the bot${NC}"
echo "================================"
