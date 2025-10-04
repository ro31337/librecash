# LibreCash Utility Scripts

This directory contains utility scripts for managing LibreCash services.

## 📁 Scripts Overview

### `up.sh` - Start Services
Starts all Docker services (PostgreSQL, RabbitMQ) and waits for them to be ready.
```bash
./bin/up.sh
```

### `down.sh` - Stop Services  
Stops all Docker services and the LibreCash bot.
```bash
./bin/down.sh
```

### `run.sh` - Run Bot
Builds and runs the LibreCash bot. Checks that all services are ready first.
```bash
./bin/run.sh
```

### `test.sh` - Run Tests
Runs the complete test suite including:
- Locale consistency (27 languages)
- Translation key validation  
- Code formatting and analysis
- Database schema validation
- Unit tests for all components
```bash
./bin/test.sh
```

### `initdb.sh` - Initialize Database
Initializes or resets the database schema. **DESTRUCTIVE** - removes all data.
```bash
./bin/initdb.sh          # Interactive mode
./bin/initdb.sh -y       # Non-interactive mode
```

## 🎯 Usage

**Recommended**: Use the main interactive menu instead:
```bash
./start.sh
```

**Direct usage**: Run scripts directly when needed:
```bash
./bin/up.sh      # Start services
./bin/initdb.sh  # Initialize database  
./bin/run.sh     # Run bot
./bin/test.sh    # Run tests
./bin/down.sh    # Stop everything
```

## 🔧 Dependencies

All scripts require:
- Docker and Docker Compose
- Go 1.19+
- Working directory: project root (`/librecash/`)

## 📊 Service Ports

- **Main Database**: localhost:5432
- **Test Database**: localhost:15433  
- **RabbitMQ**: localhost:5672
- **RabbitMQ Management**: localhost:15672 (guest/guest)

---

**Note**: These are utility scripts. Use `../start.sh` for the main interactive interface.
