package objects

import (
	"time"
)

// Exchange represents a cash/crypto exchange transaction
type Exchange struct {
	ID                int64
	UserID            int64
	ExchangeDirection string // 'cash_to_crypto' or 'crypto_to_cash'
	Status            string // 'initiated', 'posted'
	AmountUSD         *int   // nullable for now
	Lat               float64
	Lon               float64
	IsDeleted         bool       // soft delete flag
	DeletedAt         *time.Time // when exchange was deleted (nullable)
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Exchange direction constants
const (
	ExchangeDirectionCashToCrypto = "cash_to_crypto"
	ExchangeDirectionCryptoToCash = "crypto_to_cash"
)

// Exchange status constants
const (
	ExchangeStatusInitiated = "initiated"
	ExchangeStatusPosted    = "posted"
	ExchangeStatusCanceled  = "canceled"
)

// NewExchange creates a new exchange record with initial values
func NewExchange(userID int64, direction string, lat, lon float64) *Exchange {
	return &Exchange{
		UserID:            userID,
		ExchangeDirection: direction,
		Status:            ExchangeStatusInitiated,
		Lat:               lat,
		Lon:               lon,
		IsDeleted:         false,
		CreatedAt:         time.Now().UTC(), // Use UTC to avoid timezone issues
		UpdatedAt:         time.Now().UTC(), // Use UTC to avoid timezone issues
	}
}
