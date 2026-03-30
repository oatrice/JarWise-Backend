package repository

import "jarwise-backend/internal/models"

func normalizedUserID(userID string) string {
	if userID == "" {
		return models.DefaultLocalUserID
	}
	return userID
}
