package service

import (
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/repository"
)

type GraphService interface {
	GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error)
	GetExpenseGraphDataForUser(userID, jarID, period string) ([]models.GraphDataPoint, error)
}

type graphService struct {
	repo repository.TransactionRepository
}

func NewGraphService(repo repository.TransactionRepository) GraphService {
	return &graphService{repo: repo}
}

func (s *graphService) GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error) {
	return s.GetExpenseGraphDataForUser("", jarID, period)
}

func (s *graphService) GetExpenseGraphDataForUser(userID, jarID, period string) ([]models.GraphDataPoint, error) {
	// Basic validation (can be extended)
	if period != "weekly" && period != "monthly" && period != "yearly" {
		return nil, models.ErrInvalidPeriod // Needs definition or just return logic?
	}
	if userID != "" {
		return s.repo.GetExpenseGraphDataForUser(userID, jarID, period)
	}
	return s.repo.GetExpenseGraphData(jarID, period)
}
