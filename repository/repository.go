package repository

import (
	"database/sql"
	"fmt"
	"librecash/objects"
	"log"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	log.Println("[REPOSITORY] Repository initialized")
	return &Repository{db: db}
}

func (repo *Repository) FindUser(userId int64) *objects.User {
	log.Printf("[REPOSITORY] Finding user with ID: %d", userId)
	user := &objects.User{}

	var lon, lat sql.NullFloat64
	var searchRadiusKm sql.NullInt64
	var phoneNumber sql.NullString
	err := repo.db.QueryRow(
		`SELECT "userId", "menuId", "username", "firstName", "lastName", "languageCode", "lon", "lat", "search_radius_km", "phone_number"
		FROM users
		WHERE "userId" = $1
		LIMIT 1`,
		userId,
	).Scan(&user.UserId, &user.MenuId, &user.Username, &user.FirstName, &user.LastName, &user.LanguageCode, &lon, &lat, &searchRadiusKm, &phoneNumber)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[REPOSITORY] User %d not found", userId)
		} else {
			log.Printf("[REPOSITORY] Error finding user %d: %v", userId, err)
		}
		return nil
	}

	// Handle nullable location fields
	if lon.Valid {
		user.Lon = lon.Float64
	}
	if lat.Valid {
		user.Lat = lat.Float64
	}
	// Handle nullable search radius
	if searchRadiusKm.Valid {
		radius := int(searchRadiusKm.Int64)
		user.SearchRadiusKm = &radius
	}

	// Handle nullable phone number
	if phoneNumber.Valid {
		user.PhoneNumber = phoneNumber.String
	}

	log.Printf("[REPOSITORY] User %d found with language: %s", userId, user.LanguageCode)
	return user
}

func (repo *Repository) SaveUser(user *objects.User) error {
	log.Printf("[REPOSITORY] Saving user %d (username: %s, language: %s, location: %f,%f)",
		user.UserId, user.Username, user.LanguageCode, user.Lon, user.Lat)

	// Use sql.NullFloat64 for location values
	var lon, lat sql.NullFloat64
	if user.Lon != 0 || user.Lat != 0 {
		lon = sql.NullFloat64{Float64: user.Lon, Valid: true}
		lat = sql.NullFloat64{Float64: user.Lat, Valid: true}
	}

	// Handle nullable search radius
	var searchRadius interface{}
	if user.SearchRadiusKm != nil {
		searchRadius = *user.SearchRadiusKm
	}

	// First insert/update the user without geog
	_, err := repo.db.Exec(
		`INSERT INTO users ("userId", "menuId", "username", "firstName", "lastName", "languageCode", "lon", "lat", "search_radius_km", "phone_number")
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT ("userId") DO UPDATE
			SET "menuId" = $2,
			    "username" = $3,
			    "firstName" = $4,
			    "lastName" = $5,
			    "languageCode" = $6,
			    "lon" = $7,
			    "lat" = $8,
			    "search_radius_km" = $9,
			    "phone_number" = $10`,
		user.UserId, user.MenuId, user.Username, user.FirstName, user.LastName, user.LanguageCode, lon, lat, searchRadius, user.PhoneNumber,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error saving user %d: %v", user.UserId, err)
		return err
	}

	// Then update geog field if we have coordinates
	if lon.Valid && lat.Valid {
		_, err = repo.db.Exec(
			`UPDATE users SET "geog" = ST_MakePoint($1, $2)::geography WHERE "userId" = $3`,
			lon.Float64, lat.Float64, user.UserId,
		)
		if err != nil {
			log.Printf("[REPOSITORY] Error updating geog for user %d: %v", user.UserId, err)
			return err
		}
	} else {
		// Clear geog if no coordinates
		_, err = repo.db.Exec(
			`UPDATE users SET "geog" = NULL WHERE "userId" = $1`,
			user.UserId,
		)
		if err != nil {
			log.Printf("[REPOSITORY] Error clearing geog for user %d: %v", user.UserId, err)
			return err
		}
	}

	log.Printf("[REPOSITORY] User %d saved successfully", user.UserId)
	return nil
}

func (repo *Repository) ShowCallout(userId int64, featureName string) bool {
	log.Printf("[REPOSITORY] Checking callout '%s' for user %d", featureName, userId)

	var count int
	err := repo.db.QueryRow(
		`SELECT COUNT(*) FROM dismissed_feature_callouts 
		WHERE "userId" = $1 AND "featureName" = $2`,
		userId, featureName,
	).Scan(&count)

	if err != nil {
		log.Printf("[REPOSITORY] Error checking callout: %v", err)
		return true // Show callout on error
	}

	shouldShow := count == 0
	log.Printf("[REPOSITORY] Callout '%s' for user %d: show=%v", featureName, userId, shouldShow)
	return shouldShow
}

func (repo *Repository) DismissCallout(userId int64, featureName string) error {
	log.Printf("[REPOSITORY] Dismissing callout '%s' for user %d", featureName, userId)

	_, err := repo.db.Exec(
		`INSERT INTO dismissed_feature_callouts ("userId", "featureName")
		VALUES ($1, $2)
		ON CONFLICT ("userId", "featureName") DO NOTHING`,
		userId, featureName,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error dismissing callout: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Callout '%s' dismissed for user %d", featureName, userId)
	return nil
}

// UpdateUserLocation updates the user's location and PostGIS geography column
// Returns true if historical fanout should be triggered (location changed for existing user with search radius)
func (repo *Repository) UpdateUserLocation(userId int64, lon, lat float64) (bool, error) {
	log.Printf("[REPOSITORY] Updating location for user %d: lon=%f, lat=%f", userId, lon, lat)

	// Get old coordinates before updating (PRD011 requirement)
	oldUser := repo.FindUser(userId)
	var oldLat, oldLon float64
	var hasOldLocation bool
	if oldUser != nil {
		oldLat = oldUser.Lat
		oldLon = oldUser.Lon
		hasOldLocation = (oldLat != 0 || oldLon != 0)
	}

	// Update location
	_, err := repo.db.Exec(
		`UPDATE users
		SET "lon" = $2,
		    "lat" = $3,
		    "geog" = ST_MakePoint($2, $3)::geography
		WHERE "userId" = $1`,
		userId, lon, lat,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating location for user %d: %v", userId, err)
		return false, err
	}

	// Check if coordinates actually changed (PRD011 requirement)
	locationChanged := !hasOldLocation || oldLat != lat || oldLon != lon

	log.Printf("[REPOSITORY] Location updated successfully for user %d (changed: %v)", userId, locationChanged)

	// Trigger historical fanout if location changed AND user has search radius (PRD011 requirement)
	shouldTriggerHistoricalFanout := locationChanged && oldUser != nil && oldUser.SearchRadiusKm != nil

	if shouldTriggerHistoricalFanout {
		log.Printf("[REPOSITORY] Location changed for user %d with search radius, historical fanout should be triggered", userId)
	}

	return shouldTriggerHistoricalFanout, nil
}

// UpdateUserSearchRadius updates the user's search radius preference
func (repo *Repository) UpdateUserSearchRadius(userId int64, radiusKm int) error {
	log.Printf("[REPOSITORY] Updating search radius for user %d: %d km", userId, radiusKm)

	_, err := repo.db.Exec(
		`UPDATE users 
		SET "search_radius_km" = $2
		WHERE "userId" = $1`,
		userId, radiusKm,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating search radius for user %d: %v", userId, err)
		return err
	}

	log.Printf("[REPOSITORY] Search radius updated successfully for user %d", userId)
	return nil
}

// CreateExchange creates a new exchange history record
func (repo *Repository) CreateExchange(exchange *objects.Exchange) error {
	log.Printf("[REPOSITORY] Creating exchange for user %d: direction=%s, status=%s",
		exchange.UserID, exchange.ExchangeDirection, exchange.Status)

	// Set timestamps if not already set (use UTC to avoid timezone issues)
	if exchange.CreatedAt.IsZero() {
		exchange.CreatedAt = time.Now().UTC()
	}
	if exchange.UpdatedAt.IsZero() {
		exchange.UpdatedAt = time.Now().UTC()
	}

	err := repo.db.QueryRow(
		`INSERT INTO exchanges (user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		exchange.UserID, exchange.ExchangeDirection, exchange.Status, exchange.AmountUSD,
		exchange.Lat, exchange.Lon, exchange.IsDeleted, exchange.CreatedAt, exchange.UpdatedAt,
	).Scan(&exchange.ID)

	if err != nil {
		log.Printf("[REPOSITORY] Error creating exchange: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Exchange created successfully with ID: %d", exchange.ID)
	return nil
}

// GetExchangeByID retrieves an exchange record by ID (only non-deleted)
func (repo *Repository) GetExchangeByID(id int64) (*objects.Exchange, error) {
	log.Printf("[REPOSITORY] Getting exchange by ID: %d", id)

	exchange := &objects.Exchange{}
	var amountUSD sql.NullInt64
	var deletedAt sql.NullTime

	err := repo.db.QueryRow(
		`SELECT id, user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, deleted_at, created_at, updated_at
		FROM exchanges
		WHERE id = $1 AND is_deleted = FALSE`,
		id,
	).Scan(&exchange.ID, &exchange.UserID, &exchange.ExchangeDirection, &exchange.Status,
		&amountUSD, &exchange.Lat, &exchange.Lon, &exchange.IsDeleted, &deletedAt,
		&exchange.CreatedAt, &exchange.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[REPOSITORY] Exchange %d not found or deleted", id)
			return nil, nil
		}
		log.Printf("[REPOSITORY] Error getting exchange %d: %v", id, err)
		return nil, err
	}

	// Handle nullable amount
	if amountUSD.Valid {
		amount := int(amountUSD.Int64)
		exchange.AmountUSD = &amount
	}

	// Handle nullable deleted_at
	if deletedAt.Valid {
		exchange.DeletedAt = &deletedAt.Time
	}

	log.Printf("[REPOSITORY] Exchange %d found", id)
	return exchange, nil
}

// GetUserExchanges retrieves all non-deleted exchanges for a specific user
func (repo *Repository) GetUserExchanges(userID int64) ([]*objects.Exchange, error) {
	log.Printf("[REPOSITORY] Getting exchanges for user: %d", userID)

	rows, err := repo.db.Query(
		`SELECT id, user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, deleted_at, created_at, updated_at
		FROM exchanges
		WHERE user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		log.Printf("[REPOSITORY] Error getting user exchanges: %v", err)
		return nil, err
	}
	defer rows.Close()

	var exchanges []*objects.Exchange
	for rows.Next() {
		exchange := &objects.Exchange{}
		var amountUSD sql.NullInt64
		var deletedAt sql.NullTime

		err := rows.Scan(&exchange.ID, &exchange.UserID, &exchange.ExchangeDirection, &exchange.Status,
			&amountUSD, &exchange.Lat, &exchange.Lon, &exchange.IsDeleted, &deletedAt,
			&exchange.CreatedAt, &exchange.UpdatedAt)
		if err != nil {
			log.Printf("[REPOSITORY] Error scanning exchange row: %v", err)
			continue
		}

		// Handle nullable amount
		if amountUSD.Valid {
			amount := int(amountUSD.Int64)
			exchange.AmountUSD = &amount
		}

		// Handle nullable deleted_at
		if deletedAt.Valid {
			exchange.DeletedAt = &deletedAt.Time
		}

		exchanges = append(exchanges, exchange)
	}

	log.Printf("[REPOSITORY] Found %d exchanges for user %d", len(exchanges), userID)
	return exchanges, nil
}

// UpdateExchangeStatus updates the status of an exchange
func (repo *Repository) UpdateExchangeStatus(id int64, status string) error {
	log.Printf("[REPOSITORY] Updating exchange %d status to: %s", id, status)

	_, err := repo.db.Exec(
		`UPDATE exchanges
		SET status = $2, updated_at = $3
		WHERE id = $1`,
		id, status, time.Now(),
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating exchange status: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Exchange %d status updated successfully", id)
	return nil
}

// GetLastUserExchange retrieves the most recent non-deleted exchange record for a user
func (repo *Repository) GetLastUserExchange(userID int64) (*objects.Exchange, error) {
	log.Printf("[REPOSITORY] Getting last exchange for user %d", userID)

	var exchange objects.Exchange
	var amountUSD sql.NullInt64
	var deletedAt sql.NullTime

	err := repo.db.QueryRow(
		`SELECT id, user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, deleted_at, created_at, updated_at
		FROM exchanges
		WHERE user_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC
		LIMIT 1`,
		userID,
	).Scan(&exchange.ID, &exchange.UserID, &exchange.ExchangeDirection, &exchange.Status,
		&amountUSD, &exchange.Lat, &exchange.Lon, &exchange.IsDeleted, &deletedAt,
		&exchange.CreatedAt, &exchange.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[REPOSITORY] No active exchange found for user %d", userID)
			return nil, nil
		}
		log.Printf("[REPOSITORY] Error getting last exchange: %v", err)
		return nil, err
	}

	// Handle nullable amount
	if amountUSD.Valid {
		amount := int(amountUSD.Int64)
		exchange.AmountUSD = &amount
	}

	// Handle nullable deleted_at
	if deletedAt.Valid {
		exchange.DeletedAt = &deletedAt.Time
	}

	log.Printf("[REPOSITORY] Found last exchange ID %d for user %d", exchange.ID, userID)
	return &exchange, nil
}

// UpdateExchange updates an existing exchange record
func (repo *Repository) UpdateExchange(exchange *objects.Exchange) error {
	log.Printf("[REPOSITORY] Updating exchange ID %d", exchange.ID)

	var amountUSD sql.NullInt64
	if exchange.AmountUSD != nil {
		amountUSD = sql.NullInt64{Int64: int64(*exchange.AmountUSD), Valid: true}
	}

	var deletedAt sql.NullTime
	if exchange.DeletedAt != nil {
		deletedAt = sql.NullTime{Time: *exchange.DeletedAt, Valid: true}
	}

	exchange.UpdatedAt = time.Now()

	_, err := repo.db.Exec(
		`UPDATE exchanges
		SET exchange_direction = $2, status = $3, amount_usd = $4,
		    lat = $5, lon = $6, is_deleted = $7, deleted_at = $8, updated_at = $9
		WHERE id = $1`,
		exchange.ID, exchange.ExchangeDirection, exchange.Status, amountUSD,
		exchange.Lat, exchange.Lon, exchange.IsDeleted, deletedAt, exchange.UpdatedAt,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating exchange: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Exchange %d updated successfully", exchange.ID)
	return nil
}

// SoftDeleteExchange marks an exchange as deleted
func (repo *Repository) SoftDeleteExchange(exchangeID int64) error {
	log.Printf("[REPOSITORY] Soft deleting exchange %d", exchangeID)

	_, err := repo.db.Exec(
		`UPDATE exchanges
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_deleted = FALSE`,
		exchangeID,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error soft deleting exchange: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Exchange %d soft deleted successfully", exchangeID)
	return nil
}

// GetActiveExchanges retrieves all non-deleted exchanges
func (repo *Repository) GetActiveExchanges() ([]*objects.Exchange, error) {
	log.Printf("[REPOSITORY] Getting all active exchanges")

	rows, err := repo.db.Query(
		`SELECT id, user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, deleted_at, created_at, updated_at
		FROM exchanges
		WHERE is_deleted = FALSE
		ORDER BY created_at DESC`,
	)
	if err != nil {
		log.Printf("[REPOSITORY] Error getting active exchanges: %v", err)
		return nil, err
	}
	defer rows.Close()

	var exchanges []*objects.Exchange
	for rows.Next() {
		exchange := &objects.Exchange{}
		var amountUSD sql.NullInt64
		var deletedAt sql.NullTime

		err := rows.Scan(&exchange.ID, &exchange.UserID, &exchange.ExchangeDirection, &exchange.Status,
			&amountUSD, &exchange.Lat, &exchange.Lon, &exchange.IsDeleted, &deletedAt,
			&exchange.CreatedAt, &exchange.UpdatedAt)
		if err != nil {
			log.Printf("[REPOSITORY] Error scanning active exchange row: %v", err)
			continue
		}

		// Handle nullable amount
		if amountUSD.Valid {
			amount := int(amountUSD.Int64)
			exchange.AmountUSD = &amount
		}

		// Handle nullable deleted_at
		if deletedAt.Valid {
			exchange.DeletedAt = &deletedAt.Time
		}

		exchanges = append(exchanges, exchange)
	}

	log.Printf("[REPOSITORY] Found %d active exchanges", len(exchanges))
	return exchanges, nil
}

// Timeline Records Methods

// CreateTimelineRecord creates a new timeline record
func (repo *Repository) CreateTimelineRecord(record *objects.TimelineRecord) error {
	log.Printf("[REPOSITORY] Creating timeline record for exchange %d, recipient %d",
		record.ExchangeID, record.RecipientUserID)

	err := repo.db.QueryRow(
		`INSERT INTO timeline_records (exchange_id, recipient_user_id, telegram_message_id, status, is_deleted, deleted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		record.ExchangeID, record.RecipientUserID, record.TelegramMessageID, record.Status,
		record.IsDeleted, record.DeletedAt, record.CreatedAt, record.UpdatedAt,
	).Scan(&record.ID)

	if err != nil {
		log.Printf("[REPOSITORY] Error creating timeline record: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Timeline record created successfully with ID: %d", record.ID)
	return nil
}

// GetTimelineRecordsByExchange retrieves all timeline records for an exchange
func (repo *Repository) GetTimelineRecordsByExchange(exchangeID int64) ([]*objects.TimelineRecord, error) {
	log.Printf("[REPOSITORY] Getting timeline records for exchange: %d", exchangeID)

	rows, err := repo.db.Query(
		`SELECT id, exchange_id, recipient_user_id, telegram_message_id, status, is_deleted, deleted_at, created_at, updated_at
		FROM timeline_records
		WHERE exchange_id = $1
		ORDER BY created_at DESC`,
		exchangeID,
	)
	if err != nil {
		log.Printf("[REPOSITORY] Error getting timeline records: %v", err)
		return nil, err
	}
	defer rows.Close()

	var records []*objects.TimelineRecord
	for rows.Next() {
		record := &objects.TimelineRecord{}
		var telegramMessageID sql.NullInt64
		var deletedAt sql.NullTime

		err := rows.Scan(&record.ID, &record.ExchangeID, &record.RecipientUserID,
			&telegramMessageID, &record.Status, &record.IsDeleted, &deletedAt,
			&record.CreatedAt, &record.UpdatedAt)
		if err != nil {
			log.Printf("[REPOSITORY] Error scanning timeline record: %v", err)
			continue
		}

		// Handle nullable fields
		if telegramMessageID.Valid {
			msgID := int(telegramMessageID.Int64)
			record.TelegramMessageID = &msgID
		}
		if deletedAt.Valid {
			record.DeletedAt = &deletedAt.Time
		}

		records = append(records, record)
	}

	log.Printf("[REPOSITORY] Found %d timeline records for exchange %d", len(records), exchangeID)
	return records, nil
}

// UpdateTimelineRecord updates a timeline record with Telegram message ID and status
func (repo *Repository) UpdateTimelineRecord(id int64, telegramMessageID int, status string) error {
	log.Printf("[REPOSITORY] Updating timeline record %d with message ID %d, status %s",
		id, telegramMessageID, status)

	_, err := repo.db.Exec(
		`UPDATE timeline_records
		SET telegram_message_id = $2, status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		id, telegramMessageID, status,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating timeline record: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Timeline record %d updated successfully", id)
	return nil
}

// UpdateTimelineRecordStatus updates only the status of a timeline record
func (repo *Repository) UpdateTimelineRecordStatus(id int64, status string) error {
	log.Printf("[REPOSITORY] Updating timeline record %d status to %s", id, status)

	_, err := repo.db.Exec(
		`UPDATE timeline_records
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		id, status,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating timeline record status: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Timeline record %d status updated successfully", id)
	return nil
}

// MarkTimelineRecordsAsDeleted soft deletes all timeline records for an exchange
func (repo *Repository) MarkTimelineRecordsAsDeleted(exchangeID int64) error {
	log.Printf("[REPOSITORY] Marking timeline records as deleted for exchange %d", exchangeID)

	_, err := repo.db.Exec(
		`UPDATE timeline_records
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE exchange_id = $1 AND is_deleted = FALSE`,
		exchangeID,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error marking timeline records as deleted: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Timeline records marked as deleted for exchange %d", exchangeID)
	return nil
}

// SoftDeleteExchangeTimeline soft deletes all timeline records for an exchange
// This is an alias for MarkTimelineRecordsAsDeleted to match PRD009 specification
func (repo *Repository) SoftDeleteExchangeTimeline(exchangeID int64) error {
	return repo.MarkTimelineRecordsAsDeleted(exchangeID)
}

// GetActiveTimelineRecordsByExchange retrieves only non-deleted timeline records for an exchange
func (repo *Repository) GetActiveTimelineRecordsByExchange(exchangeID int64) ([]*objects.TimelineRecord, error) {
	log.Printf("[REPOSITORY] Getting active timeline records for exchange: %d", exchangeID)

	rows, err := repo.db.Query(
		`SELECT id, exchange_id, recipient_user_id, telegram_message_id, status, is_deleted, deleted_at, created_at, updated_at
		FROM timeline_records
		WHERE exchange_id = $1 AND is_deleted = FALSE
		ORDER BY created_at DESC`,
		exchangeID,
	)
	if err != nil {
		log.Printf("[REPOSITORY] Error getting active timeline records: %v", err)
		return nil, err
	}
	defer rows.Close()

	var records []*objects.TimelineRecord
	for rows.Next() {
		record := &objects.TimelineRecord{}
		var telegramMessageID sql.NullInt64
		var deletedAt sql.NullTime

		err := rows.Scan(&record.ID, &record.ExchangeID, &record.RecipientUserID,
			&telegramMessageID, &record.Status, &record.IsDeleted, &deletedAt,
			&record.CreatedAt, &record.UpdatedAt)
		if err != nil {
			log.Printf("[REPOSITORY] Error scanning active timeline record: %v", err)
			continue
		}

		// Handle nullable fields
		if telegramMessageID.Valid {
			msgID := int(telegramMessageID.Int64)
			record.TelegramMessageID = &msgID
		}
		if deletedAt.Valid {
			record.DeletedAt = &deletedAt.Time
		}

		records = append(records, record)
	}

	log.Printf("[REPOSITORY] Found %d active timeline records for exchange %d", len(records), exchangeID)
	return records, nil
}

// User Proximity Methods

// FindUsersInRadius finds all users within specified radius of given coordinates
func (repo *Repository) FindUsersInRadius(lat, lon float64, radiusKm int) ([]*objects.User, error) {
	log.Printf("[REPOSITORY] Finding users within %d km of coordinates (%f, %f)",
		radiusKm, lat, lon)

	query := `
		SELECT "userId", "menuId", "username", "firstName", "lastName", "languageCode", "lon", "lat", "search_radius_km", "phone_number"
		FROM users
		WHERE "geog" IS NOT NULL
		AND ST_DWithin("geog", ST_MakePoint($1, $2)::geography, $3 * 1000)
		ORDER BY ST_Distance("geog", ST_MakePoint($1, $2)::geography)
	`

	rows, err := repo.db.Query(query, lon, lat, radiusKm)
	if err != nil {
		log.Printf("[REPOSITORY] Error finding users in radius: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []*objects.User
	for rows.Next() {
		user := &objects.User{}
		var lon, lat sql.NullFloat64
		var searchRadiusKm sql.NullInt64
		var phoneNumber sql.NullString

		err := rows.Scan(&user.UserId, &user.MenuId, &user.Username, &user.FirstName,
			&user.LastName, &user.LanguageCode, &lon, &lat, &searchRadiusKm, &phoneNumber)
		if err != nil {
			log.Printf("[REPOSITORY] Error scanning user in radius: %v", err)
			continue
		}

		// Handle nullable location fields
		if lon.Valid {
			user.Lon = lon.Float64
		}
		if lat.Valid {
			user.Lat = lat.Float64
		}
		// Handle nullable search radius
		if searchRadiusKm.Valid {
			radius := int(searchRadiusKm.Int64)
			user.SearchRadiusKm = &radius
		}
		// Handle nullable phone number
		if phoneNumber.Valid {
			user.PhoneNumber = phoneNumber.String
		}

		users = append(users, user)
	}

	log.Printf("[REPOSITORY] Found %d users within %d km", len(users), radiusKm)
	return users, nil
}

// Contact Request Methods

// CheckContactRequestExists checks if user already requested contact for this exchange
func (repo *Repository) CheckContactRequestExists(exchangeID, requesterUserID int64) (bool, error) {
	log.Printf("[REPOSITORY] Checking if contact request exists: exchange=%d, requester=%d", exchangeID, requesterUserID)

	var count int
	err := repo.db.QueryRow(
		`SELECT COUNT(*) FROM contact_requests
		 WHERE exchange_id = $1 AND requester_user_id = $2`,
		exchangeID, requesterUserID,
	).Scan(&count)

	if err != nil {
		log.Printf("[REPOSITORY] Error checking contact request existence: %v", err)
		return false, err
	}

	exists := count > 0
	log.Printf("[REPOSITORY] Contact request exists: %t", exists)
	return exists, nil
}

// CreateContactRequest creates a new contact request record with duplicate prevention
func (repo *Repository) CreateContactRequest(exchangeID, requesterUserID int64, username, firstName, lastName string) error {
	log.Printf("[REPOSITORY] Creating contact request: exchange=%d, requester=%d, username=%s",
		exchangeID, requesterUserID, username)

	// Use transaction with SELECT FOR UPDATE to prevent race conditions
	tx, err := repo.db.Begin()
	if err != nil {
		log.Printf("[REPOSITORY] Error starting transaction: %v", err)
		return err
	}
	defer tx.Rollback() // Will be ignored if Commit() succeeds

	// Check for existing request with row-level lock
	// Use SELECT id instead of COUNT(*) because FOR UPDATE doesn't work with aggregates
	var existingID sql.NullInt64
	err = tx.QueryRow(
		`SELECT id FROM contact_requests
		 WHERE exchange_id = $1 AND requester_user_id = $2
		 FOR UPDATE`,
		exchangeID, requesterUserID,
	).Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[REPOSITORY] Error checking existing contact request: %v", err)
		return err
	}

	if existingID.Valid {
		log.Printf("[REPOSITORY] Contact request already exists (ID: %d), skipping insert", existingID.Int64)
		return tx.Commit() // Not an error, just a duplicate
	}

	// Insert new contact request
	_, err = tx.Exec(
		`INSERT INTO contact_requests (exchange_id, requester_user_id, requester_username, requester_first_name, requester_last_name)
		 VALUES ($1, $2, $3, $4, $5)`,
		exchangeID, requesterUserID, username, firstName, lastName,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error creating contact request: %v", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("[REPOSITORY] Error committing contact request transaction: %v", err)
		return err
	}

	log.Printf("[REPOSITORY] Contact request created successfully")
	return nil
}

// CountUsersInRadius counts users within specified radius of given coordinates
func (repo *Repository) CountUsersInRadius(lat, lon float64, radiusKm int) (int, error) {
	log.Printf("[REPOSITORY] Counting users within %d km of coordinates (%f, %f)",
		radiusKm, lat, lon)

	query := `
		SELECT COUNT(*)
		FROM users
		WHERE "geog" IS NOT NULL
		AND ST_DWithin("geog", ST_MakePoint($1, $2)::geography, $3 * 1000)
	`

	var count int
	err := repo.db.QueryRow(query, lon, lat, radiusKm).Scan(&count)
	if err != nil {
		log.Printf("[REPOSITORY] Error counting users in radius: %v", err)
		return 0, err
	}

	log.Printf("[REPOSITORY] Found %d users within %d km", count, radiusKm)
	return count, nil
}

// FindHistoricalExchangesInRadius finds historical active exchanges in radius for new location users
func (repo *Repository) FindHistoricalExchangesInRadius(lat, lon float64, radiusKm int, excludeUserID int64) ([]*objects.Exchange, error) {
	log.Printf("[REPOSITORY] Finding historical exchanges within %d km of coordinates (%f, %f), excluding user %d",
		radiusKm, lat, lon, excludeUserID)

	// Try different time periods to get up to 10 exchanges (max 30 days)
	timePeriods := []struct {
		days int
		name string
	}{
		{3, "3 days"},
		{7, "1 week"},
		{14, "2 weeks"},
		{30, "30 days"},
	}

	for _, period := range timePeriods {
		log.Printf("[REPOSITORY] Searching for historical exchanges in last %s", period.name)

		query := `
			WITH ranked_exchanges AS (
				SELECT e.*,
					   ROW_NUMBER() OVER (PARTITION BY e.user_id ORDER BY e.created_at DESC) as rn
				FROM exchanges e
				WHERE e.is_deleted = FALSE
				  AND e.status = 'posted'
				  AND e.user_id != $4
				  AND e.created_at >= NOW() - INTERVAL '%d days'
				  AND ST_DWithin(ST_MakePoint(e.lon, e.lat)::geography, ST_MakePoint($1, $2)::geography, $3 * 1000)
			)
			SELECT id, user_id, exchange_direction, status, amount_usd, lat, lon, is_deleted, deleted_at, created_at, updated_at
			FROM ranked_exchanges
			WHERE rn = 1
			ORDER BY created_at ASC
			LIMIT 10
		`

		formattedQuery := fmt.Sprintf(query, period.days)
		rows, err := repo.db.Query(formattedQuery, lon, lat, radiusKm, excludeUserID)
		if err != nil {
			log.Printf("[REPOSITORY] Error finding historical exchanges: %v", err)
			return nil, err
		}
		defer rows.Close()

		var exchanges []*objects.Exchange
		for rows.Next() {
			exchange := &objects.Exchange{}
			var amountUSD sql.NullInt64
			var deletedAt sql.NullTime

			err := rows.Scan(&exchange.ID, &exchange.UserID, &exchange.ExchangeDirection, &exchange.Status,
				&amountUSD, &exchange.Lat, &exchange.Lon, &exchange.IsDeleted, &deletedAt,
				&exchange.CreatedAt, &exchange.UpdatedAt)
			if err != nil {
				log.Printf("[REPOSITORY] Error scanning historical exchange row: %v", err)
				continue
			}

			// Handle nullable amount
			if amountUSD.Valid {
				amount := int(amountUSD.Int64)
				exchange.AmountUSD = &amount
			}

			// Handle nullable deleted_at
			if deletedAt.Valid {
				exchange.DeletedAt = &deletedAt.Time
			}

			exchanges = append(exchanges, exchange)
		}

		log.Printf("[REPOSITORY] Found %d historical exchanges in last %s", len(exchanges), period.name)

		// If we found enough exchanges (or any), return them
		if len(exchanges) >= 10 || len(exchanges) > 0 {
			log.Printf("[REPOSITORY] Returning %d historical exchanges", len(exchanges))
			return exchanges, nil
		}
	}

	// If no exchanges found in any time period
	log.Printf("[REPOSITORY] No historical exchanges found in any time period")
	return []*objects.Exchange{}, nil
}

// CreateLocationHistory creates a new location history record
func (repo *Repository) CreateLocationHistory(userID int64, radiusKm int) error {
	log.Printf("[REPOSITORY] Creating location history for user %d with radius %d km", userID, radiusKm)

	_, err := repo.db.Exec(
		`INSERT INTO location_histories (user_id, radius_km, lat, lon)
		 VALUES ($1, $2, 0, 0)`,
		userID, radiusKm,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error creating location history for user %d: %v", userID, err)
		return err
	}

	log.Printf("[REPOSITORY] Location history created successfully for user %d", userID)
	return nil
}

// UpdateLocationHistory updates coordinates in the latest location history record
func (repo *Repository) UpdateLocationHistory(userID int64, lat, lon float64) error {
	log.Printf("[REPOSITORY] Updating location history for user %d: lat=%f, lon=%f", userID, lat, lon)

	result, err := repo.db.Exec(
		`UPDATE location_histories
		 SET lat = $1, lon = $2, updated_at = NOW()
		 WHERE user_id = $3 AND id = (
		     SELECT id FROM location_histories
		     WHERE user_id = $3
		     ORDER BY created_at DESC
		     LIMIT 1
		 )`,
		lat, lon, userID,
	)

	if err != nil {
		log.Printf("[REPOSITORY] Error updating location history for user %d: %v", userID, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[REPOSITORY] Error checking rows affected for user %d: %v", userID, err)
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no location history record found for user %d", userID)
	}

	log.Printf("[REPOSITORY] Location history updated successfully for user %d", userID)
	return nil
}

// ShouldTriggerHistoricalFanout checks if historical fanout should be triggered
func (repo *Repository) ShouldTriggerHistoricalFanout(userID int64) (bool, error) {
	log.Printf("[REPOSITORY] Checking if historical fanout should be triggered for user %d", userID)

	var shouldFanout bool
	err := repo.db.QueryRow(
		`WITH last_two AS (
		     SELECT radius_km, lat, lon,
		            ROW_NUMBER() OVER (ORDER BY created_at DESC) as rn
		     FROM location_histories
		     WHERE user_id = $1
		     LIMIT 2
		 )
		 SELECT CASE
		     WHEN COUNT(*) = 1 THEN true  -- первая запись = всегда fanout
		     WHEN (
		         (SELECT radius_km FROM last_two WHERE rn = 1) != (SELECT radius_km FROM last_two WHERE rn = 2) OR
		         (SELECT lat FROM last_two WHERE rn = 1) != (SELECT lat FROM last_two WHERE rn = 2) OR
		         (SELECT lon FROM last_two WHERE rn = 1) != (SELECT lon FROM last_two WHERE rn = 2)
		     ) THEN true  -- изменилось = fanout
		     ELSE false   -- не изменилось = без fanout
		 END as should_fanout
		 FROM last_two`,
		userID,
	).Scan(&shouldFanout)

	if err != nil {
		log.Printf("[REPOSITORY] Error checking historical fanout trigger for user %d: %v", userID, err)
		return false, err
	}

	log.Printf("[REPOSITORY] Historical fanout should be triggered for user %d: %v", userID, shouldFanout)
	return shouldFanout, nil
}
