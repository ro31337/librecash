# LibreCash Development Guide for OpenCode

This file provides comprehensive guidance for OpenCode and all AI models working with LibreCash code.

## 🚨 MANDATORY: Always Start and Test First! 🚨

### Before doing ANY work on LibreCash:

1. **ALWAYS run `./start.sh` first** - This starts all required services:
   - PostgreSQL database with PostGIS
   - RabbitMQ message queue
   - LibreCash Telegram bot
   - Initializes database schema
   - Checks all service health

2. **ALWAYS run `./test.sh` after changes** - This validates:
   - Locale consistency across 27 languages
   - Database schema integrity
   - Code formatting and quality
   - All unit tests
   - Service connectivity

### Quick Commands:
```bash
# Start everything (ALWAYS DO THIS FIRST!)
./start.sh

# Run all tests (ALWAYS DO THIS AFTER CHANGES!)
./test.sh

# Stop everything
pkill -f librecash_bot && docker compose down

# Check status
make status

# View logs
docker compose logs -f
```

### Common Issues:
- If services fail to start: Check Docker is running
- If database errors: Run `./start.sh` to reinitialize schema
- If locale tests fail: All 27 languages must have matching keys
- If bot doesn't respond: Check logs and restart with `./start.sh`

## ⚠️ CRITICAL PROJECT CONTEXT ⚠️

### 🚨 THIS IS THE LibreCash PROJECT 🚨

**IMPORTANT BOUNDARIES:**
- **LibreCash** is the project being developed in this repository
- Focus on building LibreCash as an independent codebase

### ✅ CORRECT APPROACH
- **DO** create new LibreCash code in the root directory or new subdirectories
- **DO** build LibreCash as a separate, independent project

---

## Project Overview

**LibreCash** is a Telegram bot project for cash/payment-related services. This is a greenfield development effort.

## Architecture Patterns

### Core Architecture Patterns to Consider

1. **Message Queue Architecture**
   - Producer/Consumer pattern using RabbitMQ
   - Asynchronous message processing with priority queues
   - Separate goroutines for producing and consuming messages

2. **State-Based Menu System**
   - Users have menu states that determine interaction flow
   - Menu handlers process different states
   - State transitions managed through repository pattern

3. **Context Pattern**
   - Central context object containing dependencies
   - Passed through application layers
   - Contains: Bot API, Repository, Message Queue clients, Config

4. **Database Design**
   - PostgreSQL with PostGIS for geospatial features
   - User state management
   - Feature callout/notification tracking

5. **Configuration Management**
   - YAML-based configuration using Viper
   - Environment-specific settings
   - Telegram bot token and channel IDs

## Technology Stack
- **Language**: Go
- **Bot Framework**: telegram-bot-api
- **Database**: PostgreSQL with PostGIS
- **Message Queue**: RabbitMQ
- **Configuration**: Viper
- **Dependency Management**: Go modules (modern) or dep (legacy)
- **Containerization**: Docker with docker-compose

## Development Setup for LibreCash

When developing LibreCash, you'll need:

1. Set up Docker services (PostgreSQL, RabbitMQ)
2. Create configuration file structure
3. Implement core components following established patterns

## Important Reminders

1. **LibreCash is the PROJECT** - All new code should be for LibreCash
2. **Create NEW files** - Implement new features independently
3. **Independent Development** - LibreCash should be its own independent codebase

## User-Facing Text Rules

**NEVER add hardcoded text strings that will be shown to users in Telegram!**
- All user-visible text MUST go through the localization system (`./locales/all/`)
- Before adding ANY text that users will see, ask for permission first
- If approved, translate to ALL 27 supported languages manually
- This rule applies to: messages, buttons, confirmations, errors shown to users
- This rule does NOT apply to: log messages, debug output, internal errors

**NEVER use scripts or commands for translations!**
- NO scripts, NO echo commands, NO automation for locale files
- Edit EACH locale file manually using Read/Edit tools only
- The developer did this manually for 20 years, and you will do it too!
- Open each .po file individually and add translations by hand

## Telegram Bot UI Guidelines

**IMPORTANT: Use ONLY inline buttons!**
- ALL buttons in LibreCash MUST be inline buttons (InlineKeyboardMarkup)
- The ONLY exception is the location sharing button (Telegram limitation)
- NEVER use regular keyboard buttons (ReplyKeyboardMarkup) except for location
- Always use callback queries for inline button handling
- Remove inline buttons after they're clicked by editing the message

## Message Sending Guidelines

**CRITICAL: ALL messages MUST go through RabbitMQ!**
- NEVER send messages directly via `context.Bot.Send()`
- ALWAYS use `context.RabbitPublish.PublishTgMessage()` for ALL outgoing messages
- This prevents Telegram API rate limiting (429 errors)
- Use appropriate priority levels:
  - Priority 255: Callback answers (`AnswerCallbackQuery`) - highest priority for instant response
  - Priority 220: Normal user messages - high priority for good UX
  - Priority 200: Critical/Admin messages and message edits
  - Priority 100: Fanout notifications
  - Priority 50: Low priority messages
- NO EXCEPTIONS: Everything goes through RabbitMQ, including callback answers

## 🚨 CRITICAL RULE: Always Add Menu Transition Metrics

**EVERY time user.MenuId is changed in LibreCash code, MUST add menu transition metric!**

### Required Pattern:
```go
// Before changing MenuId
oldMenuId := user.MenuId

// Change MenuId
user.MenuId = objects.Menu_NewState

// Save user
context.Repo.SaveUser(user)

// ALWAYS add this metric after MenuId change:
metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())
```

### Import Required:
```go
import (
    "librecash/metrics"
    // ... other imports
)
```

### Where to Look for Missing Metrics:
1. **Callback handlers** - when user clicks buttons
2. **Menu transition functions** - TransitionToXXX functions  
3. **State changes** - anywhere user.MenuId = is assigned
4. **Menu handlers** - when menu logic changes state

### Files to Check:
- `menu/*.go` - all menu handlers
- `menu/main_menu.go` - button callbacks
- `menu/amount_menu.go` - amount selection
- `menu/*_menu.go` - all menu files
- Any file that imports `objects` and uses `MenuId`

### Menu ID Numbers (for reference):
- 50: US Compliance Check
- 100: Init  
- 200: Ask Location
- 250: Select Radius
- 275: Ask Phone
- 290: Historical Fanout Execute
- 295: Historical Fanout Wait
- 400: Main Menu
- 500: Amount Menu

### Example Transitions to Track:
- 400→500: Main Menu → Amount (when clicking "Есть криптовалюта")
- 500→400: Amount → Main Menu (after posting exchange)
- Any menu navigation via buttons
- Any automatic state transitions

### Why This Matters:
- Business analytics - understand user journey
- UX optimization - find drop-off points  
- A/B testing - measure menu flow changes
- Debugging - track state transition issues

## Project Documentation Standards

**PRD File Location and Naming:**
- ALL PRD files MUST be created in the `./prd/` directory
- PRD files MUST be named as `PRD00N.md` where N is the sequential number
- NO prefixes, suffixes, or descriptive names (e.g., NOT `PRD004_main_menu.md`)
- Examples: `./prd/PRD001.md`, `./prd/PRD002.md`, `./prd/PRD003.md`, `./prd/PRD004.md`

## Testing and Quality Assurance

### Running Tests
Always run `./test.sh` after making changes to ensure:
- All 27 locale files have consistent translations
- Database schema is valid
- Code formatting follows Go standards
- All unit tests pass
- Services are properly connected

### Code Quality
- Follow Go best practices and idiomatic patterns
- Ensure proper error handling
- Add appropriate logging for debugging
- Maintain consistent code style with existing codebase

## Deployment and Infrastructure

### Website Deployment
The LibreCash website is deployed using `./www/deploy.sh` which:
- Syncs files to the production server
- Restarts the Caddy container
- Verifies deployment success

### Analytics Integration
- Shynet analytics is configured for tracking
- Tracking codes are injected via Caddyfile configuration
- All pages automatically include analytics scripts

## Common Development Patterns

### Menu State Management
```go
// Example of proper menu transition
func TransitionToMainMenu(context *context.Context, user *objects.User) error {
    oldMenuId := user.MenuId
    user.MenuId = objects.Menu_Main
    if err := context.Repo.SaveUser(user); err != nil {
        return err
    }
    metrics.RecordMenuTransition(oldMenuId, user.MenuId, user.GetSupportedLanguageCode())
    return nil
}
```

### Message Sending
```go
// Correct way to send messages
func SendWelcomeMessage(context *context.Context, chatID int64) error {
    message := &messaging.TgMessage{
        ChatID: chatID,
        Text: context.Loc("welcome_message"),
        Priority: messaging.PriorityNormal,
    }
    return context.RabbitPublish.PublishTgMessage(message)
}
```

### Error Handling
```go
// Proper error handling pattern
func HandleCallback(context *context.Context, callback *tgbotapi.CallbackQuery) error {
    user, err := context.Repo.GetUser(callback.From.ID)
    if err != nil {
        log.Printf("Error getting user %d: %v", callback.From.ID, err)
        return err
    }
    
    // Process callback...
    
    return nil
}
```

## File Structure Guidelines

### Core Directories
- `menu/` - All menu handlers and UI logic
- `objects/` - Data models and database entities
- `repository/` - Database access layer
- `messaging/` - Message handling and RabbitMQ integration
- `metrics/` - Analytics and metrics collection
- `locales/all/` - Translation files for all supported languages
- `config/` - Configuration management
- `context/` - Application context and dependency injection

### Naming Conventions
- Use snake_case for file names
- Use PascalCase for Go types and functions
- Use camelCase for variables
- Menu files should end with `_menu.go`
- Test files should end with `_test.go`

This guide should be followed by all AI models and developers working on the LibreCash project to ensure consistency, quality, and proper development practices.