package domain

import "time"

// MenuOption is a node in the menu tree. A nil ParentOptionID denotes a root option.
type MenuOption struct {
	ID             int64
	ParentOptionID *int64
	Title          string
	Description    *string
	SortOrder      int
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Parent          *MenuOption
	Children        []*MenuOption
	Prompts         []*MenuOptionPrompt
	ResponseGroups  []*ResponseGroup
	ContentFeedback []*ContentFeedback
}
