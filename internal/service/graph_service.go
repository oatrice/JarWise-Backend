package service

import (
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/repository"
)

type GraphService interface {
	GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error)
}

type graphService struct {
	repo repository.TransactionRepository
}

func NewGraphService(repo repository.TransactionRepository) GraphService {
	return &graphService{repo: repo}
}

func (s *graphService) GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error) {
	// Basic validation (can be extended)
	if period != "weekly" && period != "monthly" && period != "yearly" {
		return nil, models.ErrInvalidPeriod // Needs definition or just return logic?
	}
	return s.repo.GetExpenseGraphData(jarID, period)
}
