package models

import (
	"errors"
	"time"
)

type ThumbnailKind string
type CrawlerMode string

const (
	// ThumbnailKindDefault is a small 320x240 preview
	ThumbnailKindDefault ThumbnailKind = "thumb"

	// unsupported for now
	// - ThumbnailKindLarge ThumbnailKind = "large"
	// - ThumbnailKindTall ThumbnailKind = "tall"
)

// IsKnownThumbnailKind checks if the value is supported
func (p ThumbnailKind) IsKnownThumbnailKind() bool {
	switch p {
	case
		ThumbnailKindDefault:
		return true
	}
	return false
}

func ParseThumbnailKind(str string) (ThumbnailKind, error) {
	switch str {
	case string(ThumbnailKindDefault):
		return ThumbnailKindDefault, nil
	}
	return ThumbnailKindDefault, errors.New("unknown thumbnail kind " + str)
}

// A DashboardThumbnail includes all metadata for a dashboard thumbnail
type DashboardThumbnail struct {
	Id          int64         `json:"id"`
	DashboardId int64         `json:"dashboardId"`
	PanelId     int64         `json:"panelId,omitempty"`
	Kind        ThumbnailKind `json:"kind"`
	Theme       Theme         `json:"theme"`
	Image       []byte        `json:"image"`
	MimeType    string        `json:"mimeType"`
	Updated     time.Time     `json:"updated"`
}

//
// Commands
//

// DashboardThumbnailMeta uniquely identifies a thumbnail; a natural key
type DashboardThumbnailMeta struct {
	DashboardUID string
	PanelID      int64
	Kind         ThumbnailKind
	Theme        Theme
}

type GetDashboardThumbnailCommand struct {
	DashboardThumbnailMeta

	Result *DashboardThumbnail
}

type SaveDashboardThumbnailCommand struct {
	DashboardThumbnailMeta
	Image    []byte
	MimeType string

	Result *DashboardThumbnail
}
