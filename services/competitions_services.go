package services

import (
	"api/database"
	"api/models"
	"errors"
	"fmt"
)

// Define common domain-specific errors
var (
    ErrCompetitionAccessDenied = errors.New("user does not have access to this competition")
    ErrCompetitionNotFound     = errors.New("competition not found")
    ErrDatabaseOperation       = errors.New("database operation failed")
)

// GetAccessibleCompetition fetches a competition only if the user has access to it
// It populates the competition pointer and returns an error if access is denied
func GetAccessibleCompetition(userID, competitionID string, competition *models.Competition) error {
    if userID == "" || competitionID == "" {
        return fmt.Errorf("%w: missing required parameters", ErrCompetitionAccessDenied)
    }

    // Use a more efficient query with proper indexing support
    err := database.DB.Raw(`
        SELECT DISTINCT c.*
        FROM competitions c
        JOIN competition_groups cg ON c.id = cg.competition_id
        JOIN user_groups ug ON cg.group_id = ug.group_id
        WHERE ug.user_id = ? AND c.id = ? AND c.show = true
    `, userID, competitionID).Scan(&competition).Error

    if err != nil {
        return fmt.Errorf("%w: %s", ErrDatabaseOperation, err)
    }

    // Check if we actually got a competition back
    if competition.ID == "" {
        return ErrCompetitionAccessDenied
    }

    return nil
}

// CompetitionExists checks if a competition with the given ID exists in the database
// Returns true if it exists, false otherwise or on database error
func CompetitionExists(competitionID string) bool {
    if competitionID == "" {
        return false
    }

    var count int64
    err := database.DB.Model(&models.Competition{}).
        Where("id = ?", competitionID).
        Count(&count).
        Error

    return err == nil && count > 0
}

// GetCompetitionByID fetches a competition by ID regardless of access permissions
// Useful for admin functions or internal service calls
func GetCompetitionByID(competitionID string) (models.Competition, error) {
    var competition models.Competition

    if competitionID == "" {
        return competition, fmt.Errorf("%w: missing competition ID", ErrCompetitionNotFound)
    }

    err := database.DB.Where("id = ?", competitionID).First(&competition).Error
    if err != nil {
        return competition, fmt.Errorf("%w: %s", ErrCompetitionNotFound, err)
    }

    return competition, nil
}

// GetUserCompetitions retrieves all competitions that a user has access to
func GetUserCompetitions(userID string) ([]models.Competition, error) {
    if userID == "" {
        return nil, fmt.Errorf("missing user ID")
    }

    var competitions []models.Competition
    
    err := database.DB.Raw(`
        SELECT DISTINCT c.*
        FROM competitions c
        JOIN competition_groups cg ON c.id = cg.competition_id
        JOIN user_groups ug ON cg.group_id = ug.group_id
        WHERE ug.user_id = ? AND c.show = true
        ORDER BY c.start_date DESC
    `, userID).Scan(&competitions).Error

    if err != nil {
        return nil, fmt.Errorf("%w: %s", ErrDatabaseOperation, err)
    }

    return competitions, nil
}