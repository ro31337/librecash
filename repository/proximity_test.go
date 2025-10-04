package repository

import (
	"librecash/objects"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindUsersInRadius(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	_, err := db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE "userId" IN (123456, 789012, 345678, 901234, 567890)`)
	assert.NoError(t, err)

	// Create test users at different locations
	// Central user (will be excluded from search)
	centralUser := &objects.User{
		UserId:       123456,
		Username:     "central",
		FirstName:    "Central",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7128, // NYC coordinates
		Lon:          -74.0060,
	}
	err = repo.SaveUser(centralUser)
	assert.NoError(t, err)

	// User within 5km (Manhattan)
	nearUser := &objects.User{
		UserId:       789012,
		Username:     "near",
		FirstName:    "Near",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7589, // Times Square
		Lon:          -73.9851,
	}
	err = repo.SaveUser(nearUser)
	assert.NoError(t, err)

	// User within 15km (Brooklyn)
	mediumUser := &objects.User{
		UserId:       345678,
		Username:     "medium",
		FirstName:    "Medium",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.6782, // Brooklyn
		Lon:          -73.9442,
	}
	err = repo.SaveUser(mediumUser)
	assert.NoError(t, err)

	// User far away (Philadelphia - about 150km)
	farUser := &objects.User{
		UserId:       901234,
		Username:     "far",
		FirstName:    "Far",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          39.9526, // Philadelphia
		Lon:          -75.1652,
	}
	err = repo.SaveUser(farUser)
	assert.NoError(t, err)

	// User without location (should be excluded)
	noLocationUser := &objects.User{
		UserId:       567890,
		Username:     "nolocation",
		FirstName:    "No",
		LastName:     "Location",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		// No Lat/Lon set
	}
	err = repo.SaveUser(noLocationUser)
	assert.NoError(t, err)

	// Test 6km radius - should find near user + central user (near is ~5.5km away)
	users6km, err := repo.FindUsersInRadius(centralUser.Lat, centralUser.Lon, 6)
	assert.NoError(t, err)
	assert.Len(t, users6km, 2) // near + central

	// Should be ordered by distance (nearest first)
	assert.Equal(t, centralUser.UserId, users6km[0].UserId) // central is at exact coordinates (0 distance)
	assert.Equal(t, nearUser.UserId, users6km[1].UserId)

	// Test 15km radius - should find central, near and medium users
	users15km, err := repo.FindUsersInRadius(centralUser.Lat, centralUser.Lon, 15)
	assert.NoError(t, err)
	assert.Len(t, users15km, 3) // central + near + medium

	// Should be ordered by distance (nearest first)
	assert.Equal(t, centralUser.UserId, users15km[0].UserId) // central is at exact coordinates
	assert.Equal(t, nearUser.UserId, users15km[1].UserId)
	assert.Equal(t, mediumUser.UserId, users15km[2].UserId)

	// Test 50km radius - should find central, near and medium users (far user is too far)
	users50km, err := repo.FindUsersInRadius(centralUser.Lat, centralUser.Lon, 50)
	assert.NoError(t, err)
	assert.Len(t, users50km, 3) // central + near + medium

	// Test 200km radius - should find all users with location
	users200km, err := repo.FindUsersInRadius(centralUser.Lat, centralUser.Lon, 200)
	assert.NoError(t, err)
	assert.Len(t, users200km, 4) // central + near + medium + far (but not no-location)
}

func TestCountUsersInRadius(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		t.Skip("Database tests require PostgreSQL connection")
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	var err error
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE "userId" IN (123456, 789012, 345678)`)
	assert.NoError(t, err)

	// Create test users
	centralUser := &objects.User{
		UserId:       123456,
		Username:     "central",
		FirstName:    "Central",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(centralUser)
	assert.NoError(t, err)

	nearUser := &objects.User{
		UserId:       789012,
		Username:     "near",
		FirstName:    "Near",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7589,
		Lon:          -73.9851,
	}
	err = repo.SaveUser(nearUser)
	assert.NoError(t, err)

	mediumUser := &objects.User{
		UserId:       345678,
		Username:     "medium",
		FirstName:    "Medium",
		LastName:     "User",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.6782,
		Lon:          -73.9442,
	}
	err = repo.SaveUser(mediumUser)
	assert.NoError(t, err)

	// Test counting
	count6km, err := repo.CountUsersInRadius(centralUser.Lat, centralUser.Lon, 6)
	assert.NoError(t, err)
	assert.Equal(t, 2, count6km) // central + near

	count15km, err := repo.CountUsersInRadius(centralUser.Lat, centralUser.Lon, 15)
	assert.NoError(t, err)
	assert.Equal(t, 3, count15km) // central + near + medium

	count50km, err := repo.CountUsersInRadius(centralUser.Lat, centralUser.Lon, 50)
	assert.NoError(t, err)
	assert.Equal(t, 3, count50km) // central + near + medium
}

func TestFindUsersInRadiusIncludesAllUsers(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	var err error
	_, err = db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create test users
	user1 := &objects.User{
		UserId:       123456,
		Username:     "user1",
		FirstName:    "User",
		LastName:     "One",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7128,
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user1)
	assert.NoError(t, err)

	user2 := &objects.User{
		UserId:       789012,
		Username:     "user2",
		FirstName:    "User",
		LastName:     "Two",
		LanguageCode: "en",
		MenuId:       objects.Menu_Main,
		Lat:          40.7128, // Same location as user1
		Lon:          -74.0060,
	}
	err = repo.SaveUser(user2)
	assert.NoError(t, err)

	// Search from user1's location - should find both users
	users, err := repo.FindUsersInRadius(user1.Lat, user1.Lon, 50)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Verify both users are found
	userIDs := make([]int64, len(users))
	for i, user := range users {
		userIDs[i] = user.UserId
	}
	assert.Contains(t, userIDs, user1.UserId, "Should include user1")
	assert.Contains(t, userIDs, user2.UserId, "Should include user2")
}

func TestFindUsersInRadiusWithSearchRadius(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()
	repo := NewRepository(db)

	// Clean up any existing data
	var err error
	_, err = db.Exec(`DELETE FROM contact_requests`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM exchanges`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM location_histories`)
	assert.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users`)
	assert.NoError(t, err)

	// Create user with search radius
	radius := 15
	user := &objects.User{
		UserId:         123456,
		Username:       "testuser",
		FirstName:      "Test",
		LastName:       "User",
		LanguageCode:   "en",
		MenuId:         objects.Menu_Main,
		Lat:            40.7128,
		Lon:            -74.0060,
		SearchRadiusKm: &radius,
	}
	err = repo.SaveUser(user)
	assert.NoError(t, err)

	// Find the user and verify search radius is preserved
	users, err := repo.FindUsersInRadius(40.7589, -73.9851, 50)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.NotNil(t, users[0].SearchRadiusKm)
	assert.Equal(t, 15, *users[0].SearchRadiusKm)
}
