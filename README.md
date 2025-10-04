# LibreCash - USDT/Cash Exchange Bot

LibreCash is a Telegram bot built with Go that facilitates cash-to-crypto and crypto-to-cash exchanges, with primary focus on TRC20 tokens on the Tron network. The platform leverages geolocation to connect users for peer-to-peer exchanges in their local area.

## 💱 Exchange Philosophy

While LibreCash is designed with TRC20 Tron tokens as a primary focus, the platform is fundamentally flexible and unlimited in what can be exchanged if you're willing to run your own instance and add your own changes. The bot serves as a geolocation-based matching service that connects people who want to exchange anything of value.

### How It Works
1. **Post what you have** and **what you want** (USDT or cash for this instance)
2. **Get matched** with nearby users (within your selected radius)
3. **Connect directly** to arrange the exchange
4. **Meet in person** to complete the transaction
5. **Rate and review** the experience (optional)

The platform provides the connection - you determine the terms of the exchange.

## 🚀 Quick Start

### Quick Start
```bash
# Start all services
./start.sh up

# Initialize database (first time only)
./start.sh initdb

# Run the bot
./start.sh run

# Run tests
./test.sh

# Stop all services
./start.sh down
```

### Interactive Mode
```bash
./start.sh -i
```
This opens an interactive menu for managing services.

## 📋 Prerequisites

- **Docker** and **Docker Compose**
- **Go 1.19+**
- **Telegram Bot Token** (set in configuration)

## ⚙️ Initial Setup

Before running LibreCash for the first time, you need to set up your configuration:

```bash
# Copy the example configuration file
cp librecash.yml.example librecash.yml

# Edit the configuration file with your settings
nano librecash.yml
```

### Required Configuration
- **Telegram Bot Token**: Get from @BotFather on Telegram
- **Database settings**: PostgreSQL connection details
- **RabbitMQ settings**: Message queue configuration

The configuration file is well-documented with comments explaining each setting.

## 🏗️ Architecture

### Project Structure
```
librecash/
├── start.sh          # Main management script
├── test.sh           # Test runner
├── bin/              # Utility scripts
│   ├── up.sh         # Start Docker services
│   ├── down.sh       # Stop Docker services
│   ├── run.sh        # Build and run bot
│   └── initdb.sh     # Initialize database
├── config.yml        # Configuration
├── locales/all/      # Translations (27 languages)
├── menu/             # Menu handlers
├── objects/          # Data models
├── repository/       # Database layer
├── messaging/        # Message handling
├── metrics/          # Analytics
└── main.go           # Bot entry point
```

### Services
- **PostgreSQL** with PostGIS - Main database (port 5432)
- **PostgreSQL** with PostGIS - Test database (port 15433)
- **RabbitMQ** - Message queue (port 5672, management UI: 15672)

### Components
- **Telegram Bot** - Main bot interface
- **Repository Layer** - Database operations
- **Menu System** - State-based user interactions
- **Geolocation** - PostGIS spatial queries
- **Message Queue** - Async message processing

## 🧪 Testing

All tests must pass before deployment:

```bash
./test.sh
```

Tests include:
- ✅ Locale consistency (27 languages)
- ✅ Translation key validation
- ✅ Code formatting and analysis
- ✅ Database schema validation
- ✅ Spatial index verification
- ✅ Unit tests for all components

## 🗄️ Database Management

### Initialize Database
```bash
./start.sh initdb        # Initialize database schema
```

### Database Schema
- **users** - User profiles with geolocation
- **exchanges** - Exchange requests
- **contact_requests** - Contact between users
- **timeline_records** - Exchange history

## 🌍 Localization

LibreCash supports 27 languages with complete translations:
- English, Spanish, French, German, Italian, Portuguese
- Russian, Ukrainian, Polish, Czech, Slovak, Hungarian
- Arabic, Hebrew, Turkish, Persian, Hindi, Chinese
- Japanese, Korean, Thai, Vietnamese, Indonesian
- Dutch, Swedish, Norwegian, Danish, Finnish

All user-facing text must be localized in `./locales/all/`.

## 🔧 Configuration

Edit `config.yml`:
```yaml
telegram:
  token: "YOUR_BOT_TOKEN"
  
database:
  host: "localhost"
  port: 5432
  name: "librecash"
  user: "librecash"
  password: "librecash"

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
```

## 📊 Service Status

Check service status:
```bash
./start.sh status
```

Or manually:
```bash
docker compose ps
pgrep -f librecash_bot
```

## 🐛 Troubleshooting

### Services won't start
```bash
# Check Docker
docker info

# Restart services
./start.sh down
./start.sh up
```

### Database issues
```bash
# Reinitialize database
./start.sh initdb
```

### Tests failing
```bash
# Check service status
./start.sh status

# View logs
./start.sh logs
```

### Bot not responding
```bash
# Check bot process
pgrep -f librecash_bot

# Restart bot
./start.sh run
```

## 🔗 Useful Links

- **Grafana Dashboards**: http://localhost:3000 (admin/librecash)
- **VictoriaMetrics UI**: http://localhost:8428/vmui/
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Database**: postgresql://localhost:5432/librecash
- **Test Database**: postgresql://localhost:15433/librecash_test

## 📊 Monitoring & Metrics

LibreCash includes comprehensive metrics collection for business intelligence and operational monitoring.

### Available Dashboards
- **VictoriaMetrics UI**: http://localhost:8428/vmui/ - Metrics exploration and querying
- **Grafana Dashboards**: http://localhost:3000 (admin/librecash) - Business intelligence

### Key Metrics Categories
- 🐰 RabbitMQ Message Flow - Queue throughput and success rates
- 📱 Telegram Message Delivery - API calls and error tracking  
- 🧭 Menu Transitions - User journey and conversion funnels
- 📨 Fanout Messages - Exchange notification delivery
- 📋 Listing Operations - Exchange creation/cancellation
- 📞 Contact Requests - User interaction tracking
- 🌍 Geographic Data - User location analytics

### Quick Metrics Check
```bash
# Check all LibreCash metrics
curl -s http://localhost:8081/metrics | grep librecash

# Check service status
./start.sh status
```

### Metrics Endpoints
- **Application Metrics**: http://localhost:8081/metrics
- **VictoriaMetrics UI**: http://localhost:8428/vmui/
- **VictoriaMetrics API**: http://localhost:8428/api/v1/query

## 📊 Grafana Business Intelligence Dashboards

LibreCash includes 4 pre-configured Grafana dashboards for comprehensive business intelligence and operational monitoring.

### Access Grafana
- **URL**: http://localhost:3000
- **Username**: admin
- **Password**: librecash

### Available Dashboards

#### 1. 🚀 Executive Business Dashboard
**URL**: http://localhost:3000/d/librecash-business
- **Target Audience**: CEO, Product Manager, Business Stakeholders
- **Key Metrics**: User growth, conversion funnels, geographic distribution, business activity
- **Panels**: New users today, total exchanges, conversion rates, user distribution by language

#### 2. ⚡ Operational Performance Dashboard
**URL**: http://localhost:3000/d/librecash-ops
- **Target Audience**: DevOps, SRE, Technical Team
- **Key Metrics**: System health, RabbitMQ throughput, Telegram API success rates
- **Panels**: Success rate gauges, message throughput, error tracking

#### 3. 🧭 User Journey Analytics Dashboard
**URL**: http://localhost:3000/d/librecash-journey
- **Target Audience**: Product Manager, UX Designer, Growth Team
- **Key Metrics**: Menu transitions, conversion funnels, user behavior analysis
- **Panels**: Registration funnel, location sharing rates, user flow visualization

#### 4. 💰 Real-time Activity Dashboard
**URL**: http://localhost:3000/d/librecash-realtime
- **Target Audience**: Operations Team, Customer Support
- **Key Metrics**: Live user activity, real-time system status, geographic activity
- **Panels**: Live activity feed, system status indicators, real-time statistics

### Dashboard Features
- **Auto-refresh**: Dashboards update every 1-5 seconds
- **Time ranges**: Configurable from 15 minutes to 6 hours
- **Interactive**: Click and drill-down capabilities
- **Mobile-friendly**: Responsive design for all devices
- **Zero-config**: Automatically provisioned with VictoriaMetrics datasource

## 🤖 Available Bot Commands

LibreCash bot supports the following slash commands:

### User Commands

#### `/start`
- **Purpose**: Initialize or restart the bot interaction
- **Behavior**: Begins the registration flow (US compliance → radius selection → location sharing → phone number)
- **Available from**: Any state
- **Result**: Transitions to US compliance check

#### `/location`
- **Purpose**: Update location and search radius settings
- **Behavior**: Directly jumps to radius selection menu
- **Available from**: Any state (especially useful from main menu)
- **Result**: Transitions to radius selection → location sharing → returns to main menu
- **Use case**: When user moves to a new area or wants to change search radius

#### `/language`
- **Purpose**: Change the bot's interface language
- **Behavior**: Shows language selection menu with 27 supported languages
- **Available from**: Any state
- **Result**: Displays inline buttons for all supported languages
- **Use case**: When user wants to switch to their preferred language
- **Languages**: English, Spanish, French, German, Italian, Portuguese, Russian, Ukrainian, Polish, Turkish, Arabic, Persian, Hebrew, Hindi, Chinese (Simplified/Traditional), Indonesian, Vietnamese, Thai, Burmese, Kazakh, Azerbaijani, Bulgarian, Romanian, Filipino

#### `/exchange`
- **Purpose**: Quick access to the main exchange menu
- **Behavior**: Shows fresh main menu at bottom of chat (solves "floating buttons" problem)
- **Available from**: Any state (for initialized users only)
- **Requirements**: User must have completed profile setup (location + search radius)
- **Result**: Resets to main menu with exchange options:
  - 💵 Have Cash, Need Crypto
  - ₿ Have Crypto, Need Cash
- **Error handling**: Shows setup reminder if user not initialized
- **Use case**: When exchange buttons have scrolled up due to notifications

### Command Features
- **Case-insensitive**: All commands work regardless of case
- **State preservation**: User data is preserved during command execution
- **Metrics tracking**: All commands are tracked for analytics
- **Instant response**: Commands are processed immediately
- **Validation**: Commands validate user state before execution

### Usage Examples
```
/start          # Begin registration or restart flow
/location       # Update location settings
/language       # Change interface language
/exchange       # Quick access to exchange menu
/Location       # Same as above (case-insensitive)
/LANGUAGE       # Same as above (case-insensitive)
```

### Command Error Handling
- **/exchange** requires completed profile setup
- Uninitialized users receive helpful setup reminders
- User state remains unchanged for failed commands
- All error messages are localized in user's preferred language

## 📝 Development

### Adding Features
1. Write tests first
2. Implement feature
3. Update localization (all 27 languages)
4. Run `./test.sh` to ensure all tests pass
5. Test manually with `./start.sh run`

### Code Style
- Use `go fmt` for formatting
- Follow Go conventions
- Add comprehensive tests
- Document public functions

## 🎯 Project Status

**✅ Production Ready**
- All tests passing
- Complete localization (27 languages)
- Robust error handling
- Comprehensive logging
- Docker containerization
- Advanced metrics & monitoring
- Grafana business intelligence dashboards
- Location update command
- TRC20 Tron token exchange support
- Flexible exchange platform for any goods/services
- Enterprise-grade monitoring & analytics ready

---

**LibreCash** - Making cash exchanges simple and secure! 🚀
