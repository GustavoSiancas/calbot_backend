package domain

import "time"

// MenuOptionPrompt is a selectable prompt variant associated with a menu option.
type MenuOptionPrompt struct {
	ID           int64
	MenuOptionID int64
	Message      string
	SortOrder    int
	Weight       int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time

	MenuOption *MenuOption
}
