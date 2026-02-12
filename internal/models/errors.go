package models

import "errors"

var (
	ErrInvalidPeriod = errors.New("invalid period: must be weekly, monthly, or yearly")
)
