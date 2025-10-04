#!/bin/bash

# LibreCash Management Script
# Command-line interface for managing LibreCash services
# Use -i or --interactive for interactive menu

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Function to show usage
show_usage() {
    echo -e "${BOLD}${BLUE}LibreCash Manager${NC}"
    echo ""
    echo -e "${CYAN}Usage:${NC}"
    echo "  $0 [command]                 - Execute command directly"
    echo "  $0 -i|--interactive          - Show interactive menu"
    echo "  $0 -h|--help                 - Show this help"
    echo ""
    echo -e "${CYAN}Available commands:${NC}"
    echo -e "  ${GREEN}up${NC}       - Start Docker services (PostgreSQL, RabbitMQ, VictoriaMetrics)"
    echo -e "  ${GREEN}run${NC}      - Build and run LibreCash bot"
    echo -e "  ${GREEN}initdb${NC}   - Initialize/reset database schema"
    echo -e "  ${GREEN}down${NC}     - Stop all services"
    echo ""
    echo -e "  ${YELLOW}status${NC}   - Show service status"
    echo -e "  ${YELLOW}logs${NC}     - Show service logs"
    echo -e "  ${RED}restart${NC}  - Auto-restart bot on crashes (infinite loop)"
    echo ""
    echo -e "${CYAN}Examples:${NC}"
    echo "  $0 up                        - Start services"
    echo "  $0 run                       - Run the bot"
    echo "  ./test.sh                    - Run tests"
    echo "  $0 -i                        - Interactive mode"
}

# Function to show the main menu
show_menu() {
    clear
    echo -e "${BOLD}${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${BLUE}║           LibreCash Manager          ║${NC}"
    echo -e "${BOLD}${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}Choose an option:${NC}"
    echo ""
    echo -e "  ${GREEN}1)${NC} ${BOLD}up${NC}     - Start Docker services (PostgreSQL, RabbitMQ)"
    echo -e "  ${GREEN}2)${NC} ${BOLD}run${NC}    - Build and run LibreCash bot"
    echo -e "  ${GREEN}3)${NC} ${BOLD}initdb${NC} - Initialize/reset database schema"
    echo -e "  ${GREEN}4)${NC} ${BOLD}down${NC}   - Stop all services"
    echo ""
    echo -e "  ${YELLOW}5)${NC} ${BOLD}status${NC} - Show service status"
    echo -e "  ${YELLOW}6)${NC} ${BOLD}logs${NC}   - Show service logs"
    echo ""
    echo -e "  ${RED}8)${NC} ${BOLD}restart${NC} - Auto-restart bot on crashes"
    echo ""
    echo -e "  ${RED}0)${NC} ${BOLD}exit${NC}   - Exit"
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
}

# Function to show service status
show_status() {
    echo -e "${BLUE}📊 Service Status${NC}"
    echo "================================"
    
    # Check Docker
    if docker info >/dev/null 2>&1; then
        echo -e "Docker:        ${GREEN}✅ Running${NC}"
    else
        echo -e "Docker:        ${RED}❌ Not running${NC}"
        return
    fi
    
    # Check Docker Compose services
    if docker compose ps | grep -q "Up"; then
        echo -e "Services:      ${GREEN}✅ Running${NC}"
        
        # Check individual services
        if docker compose ps db | grep -q "Up"; then
            echo -e "  Database:    ${GREEN}✅ Up${NC}"
        else
            echo -e "  Database:    ${RED}❌ Down${NC}"
        fi
        
        if docker compose ps rabbit | grep -q "Up"; then
            echo -e "  RabbitMQ:    ${GREEN}✅ Up${NC}"
        else
            echo -e "  RabbitMQ:    ${RED}❌ Down${NC}"
        fi
        
        if docker compose ps db_test | grep -q "Up"; then
            echo -e "  Test DB:     ${GREEN}✅ Up${NC}"
        else
            echo -e "  Test DB:     ${YELLOW}⚠️  Down (OK)${NC}"
        fi

        if docker compose ps victoriametrics | grep -q "Up"; then
            echo -e "  VictoriaMetrics: ${GREEN}✅ Up${NC}"
        else
            echo -e "  VictoriaMetrics: ${RED}❌ Down${NC}"
        fi
    else
        echo -e "Services:      ${RED}❌ Not running${NC}"
    fi
    
    # Check LibreCash bot
    if pgrep -f "librecash_bot" >/dev/null; then
        echo -e "Bot:           ${GREEN}✅ Running (PID: $(pgrep -f librecash_bot))${NC}"
    else
        echo -e "Bot:           ${YELLOW}⚠️  Not running${NC}"
    fi
    
    echo "================================"
}

# Function to show logs
show_logs() {
    echo -e "${BLUE}📋 Service Logs${NC}"
    echo "================================"
    echo -e "${YELLOW}Press Ctrl+C to stop viewing logs${NC}"
    echo ""
    docker compose logs -f
}

# Function to wait for user input
wait_for_input() {
    echo ""
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read
}

# Function to auto-restart bot on crashes
auto_restart_bot() {
    echo -e "${RED}${BOLD}🔄 AUTO-RESTART MODE ACTIVATED${NC}"
    echo -e "${YELLOW}Bot will automatically restart if it crashes${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop auto-restart${NC}"
    echo ""

    local restart_count=0

    # Trap Ctrl+C to exit gracefully
    trap 'echo -e "\n${GREEN}Auto-restart stopped by user${NC}"; exit 0' INT

    while true; do
        restart_count=$((restart_count + 1))

        echo -e "${CYAN}=== RESTART #${restart_count} ===${NC}"
        echo -e "${BLUE}$(date): Starting LibreCash bot...${NC}"

        # Run the bot
        ./bin/run.sh

        # If we get here, the bot crashed
        exit_code=$?
        echo -e "${RED}$(date): Bot crashed with exit code ${exit_code}${NC}"

        # Wait a bit before restarting
        echo -e "${YELLOW}Waiting 3 seconds before restart...${NC}"
        sleep 3

        echo -e "${YELLOW}Restarting bot...${NC}"
        echo ""
    done
}

# Function to execute a command
execute_command() {
    local cmd="$1"
    case $cmd in
        up)
            echo -e "${BLUE}Starting services...${NC}"
            ./bin/up.sh
            ;;
        run)
            echo -e "${BLUE}Running LibreCash bot...${NC}"
            ./bin/run.sh
            ;;

        initdb)
            echo -e "${BLUE}Initializing database...${NC}"
            ./bin/initdb.sh
            ;;
        down)
            echo -e "${BLUE}Stopping services...${NC}"
            ./bin/down.sh
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        restart)
            echo -e "${RED}Starting auto-restart mode...${NC}"
            auto_restart_bot
            ;;
        *)
            echo -e "${RED}Error: Unknown command '$cmd'${NC}"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Interactive menu loop
interactive_mode() {
    while true; do
        show_menu
        echo -n "Enter your choice [0-9]: "
        read choice

        case $choice in
            1)
                execute_command "up"
                wait_for_input
                ;;
            2)
                execute_command "run"
                wait_for_input
                ;;
            3)
                execute_command "initdb"
                wait_for_input
                ;;
            4)
                execute_command "down"
                wait_for_input
                ;;
            5)
                execute_command "status"
                wait_for_input
                ;;
            6)
                execute_command "logs"
                ;;
            8)
                execute_command "restart"
                # Note: auto_restart_bot runs in infinite loop, so we won't return here
                ;;
            0)
                echo -e "${GREEN}Goodbye!${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}Invalid option. Please try again.${NC}"
                sleep 2
                ;;
        esac
    done
}

# Main script logic
if [ $# -eq 0 ]; then
    # No arguments - show usage
    show_usage
    exit 0
elif [ "$1" = "-i" ] || [ "$1" = "--interactive" ]; then
    # Interactive mode
    interactive_mode
elif [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    # Show help
    show_usage
    exit 0
else
    # Execute command directly
    execute_command "$1"
fi
