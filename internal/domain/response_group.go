package domain

import "time"

// ResponseGroup groups the ordered response items for a menu option.
type ResponseGroup struct {
	ID           int64
	MenuOptionID int64
	Name         string
	Description  *string
	SortOrder    int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time

	MenuOption    *MenuOption
	ResponseItems []*ResponseItem
}
