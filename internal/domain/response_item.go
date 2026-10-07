package domain

import (
	"encoding/json"
	"time"
)

// ResponseItemType identifies the content rendered as part of a response group.
type ResponseItemType string

const (
	ResponseItemTypeText     ResponseItemType = "TEXT"
	ResponseItemTypeLink     ResponseItemType = "LINK"
	ResponseItemTypeImage    ResponseItemType = "IMAGE"
	ResponseItemTypeYouTube  ResponseItemType = "YOUTUBE"
	ResponseItemTypeVideo    ResponseItemType = "VIDEO"
	ResponseItemTypeDocument ResponseItemType = "DOCUMENT"
)

// ResponseItem is an ordered content element in a response group.
type ResponseItem struct {
	ID              int64
	ResponseGroupID int64
	Type            ResponseItemType
	Text            *string
	URL             *string
	Caption         *string
	Metadata        json.RawMessage
	SortOrder       int
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time

	ResponseGroup *ResponseGroup
}
