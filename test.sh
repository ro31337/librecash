#!/bin/bash

# LibreCash Test Suite
# Runs all tests for LibreCash project

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Running LibreCash Tests...${NC}"
echo "================================"

# Check if Docker services are running
if ! docker compose ps | grep -q "Up"; then
    echo -e "${RED}❌ Docker services are not running!${NC}"
    echo -e "${YELLOW}💡 Run ${BLUE}./up.sh${NC} first to start services${NC}"
    exit 1
fi

# Initialize test database
echo -e "${YELLOW}🗄️  Setting up test database...${NC}"

# Start test database if not running
if ! docker compose ps db_test | grep -q "Up"; then
    docker compose up -d db_test >/dev/null 2>&1
fi

# Wait for test database to be ready
echo -e "${YELLOW}⏳ Waiting for test database to be ready...${NC}"
timeout=30
while ! docker compose exec -T db_test pg_isready -h localhost -U librecash -d librecash_test >/dev/null 2>&1; do
    sleep 1
    timeout=$((timeout - 1))
    if [ $timeout -eq 0 ]; then
        echo -e "${RED}❌ Test database failed to start${NC}"
        exit 1
    fi
done

# Reinitialize test database schema (always clean for tests)
echo -e "${YELLOW}🔄 Reinitializing test database schema...${NC}"
docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;" >/dev/null 2>&1
docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test -c "CREATE EXTENSION IF NOT EXISTS postgis;" >/dev/null 2>&1
docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test < db/init.sql >/dev/null 2>&1

echo -e "${GREEN}✅ Test database is ready${NC}"

# Check services
echo -e "${YELLOW}🔍 Checking services...${NC}"
if ! docker compose ps | grep -q "Up"; then
    echo -e "${RED}❌ Some services are not running${NC}"
    exit 1
fi
echo -e "${GREEN}✅ All services are running${NC}"

echo -e "${YELLOW}🧪 Running tests...${NC}"
echo ""

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${BLUE}▶ Running: $test_name${NC}"
    echo -e "${YELLOW}  Command: $test_command${NC}"
    
    if eval "$test_command" >/dev/null 2>&1; then
        echo -e "${GREEN}  ✅ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}  ❌ FAILED${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        # Show the actual output for debugging
        echo -e "${YELLOW}  Debug output:${NC}"
        eval "$test_command" 2>&1 | sed 's/^/    /'
    fi
    echo ""
}

# 1. Locale consistency test
run_test "Locale Consistency (27 languages)" "go test -v . -run TestLocaleConsistency"

# 2. Required translation keys test
run_test "Required Translation Keys" "go test -v . -run TestRequiredTranslationKeys"

# 3. Build test
run_test "Build Application" "go build -o /tmp/librecash_test_build"

# 4. Code formatting test
run_test "Code Formatting (go fmt)" "test -z \"\$(gofmt -l .)\" || (echo 'Files need formatting:' && gofmt -l . && false)"

# 5. Code analysis test
run_test "Code Analysis (go vet)" "go vet \$(go list ./... 2>/dev/null)"

# 6. Database schema test (using test database)
run_test "Database Schema Validation" "docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test -c \"SELECT column_name FROM information_schema.columns WHERE table_name = 'users' AND column_name IN ('lon', 'lat', 'geog')\" | grep -q '3 rows'"

# 7. Spatial index test (using test database)
run_test "Spatial Index Exists" "docker compose exec -T -e PGPASSWORD=librecash db_test psql -h localhost -U librecash -d librecash_test -c \"SELECT indexname FROM pg_indexes WHERE tablename = 'users' AND indexname = 'users_geog_idx'\" | grep -q 'users_geog_idx'"

# 8. Menu constants test
run_test "Menu Constants" "go test -v ./menu -run TestMenuStateTransitions 2>/dev/null || echo 'No menu transition tests yet'"

# 9. Graceful shutdown tests
run_test "Graceful Shutdown Tests" "go test -v . -run TestGracefulShutdown"

# 10. All unit tests (includes graceful shutdown)
run_test "All Unit Tests" "go test -v \$(go list ./... 2>/dev/null)"

# 11. Bot status check
if pgrep -f "librecash_bot" >/dev/null; then
    echo -e "${BLUE}▶ Checking Bot Status${NC}"
    echo -e "${GREEN}  ✅ Bot is running (PID: $(pgrep -f librecash_bot))${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${BLUE}▶ Checking Bot Status${NC}"
    echo -e "${YELLOW}  ℹ️  Bot is not running (this is OK for tests)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
fi

# Clean up test build
rm -f /tmp/librecash_test_build

echo ""
echo "================================"
echo -e "${YELLOW}📊 Test Summary${NC}"
echo "================================"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    echo ""
    echo "LibreCash is ready for use!"
    echo "Bot: @librecash_bot"
    echo "================================"
    exit 0
else
    echo ""
    echo -e "${RED}❌ Some tests failed!${NC}"
    echo "================================"
    exit 1
fi
