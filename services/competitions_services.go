package services

import (
	"api/database"
	"api/models"
	"fmt"
)

// FetchCompetition get the target competition but returns an error if the user does not have access to it
func GetAccessibleCompetition(userID, competitionID string, competition *models.Competition) error {
    err := database.DB.Raw(`
        SELECT DISTINCT c.*
        FROM competitions c
        JOIN competition_groups cat ON c.id = cat.competition_id
        JOIN user_groups ug ON cat.group_id = ug.group_id
        WHERE ug.user_id = ? AND c.id = ? AND c.show = true
    `, userID, competitionID).Scan(&competition).Error

    if err != nil || competition.ID == "" {
        return fmt.Errorf("user does not have access to this competition")
    }

	return nil
}