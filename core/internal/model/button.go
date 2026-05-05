package model

type DisplayMode string

const (
	DisplayModeLive           DisplayMode = "live"
	DisplayModeBlackout       DisplayMode = "blackout"
	DisplayModeFreezeTracking DisplayMode = "freeze_tracking"
)

func (d DisplayMode) IsValid() bool {
	switch d {
	case DisplayModeLive, DisplayModeBlackout, DisplayModeFreezeTracking:
		return true
	default:
		return false
	}
}

type Button struct {
	ID              int64       `db:"id" json:"id"`
	Label           string      `db:"label" json:"label"`
	TextColor       string      `db:"text_color" json:"textColor"`
	BackgroundColor string      `db:"background_color" json:"backgroundColor"`
	Icon            *string     `db:"icon" json:"icon,omitempty"`
	DisplayMode     DisplayMode `db:"display_mode" json:"displayMode"`
	Position        int         `db:"position" json:"position"`
	IsDeletable     bool        `db:"is_deletable" json:"isDeletable"`
}

func (b *Button) IsSystem() bool {
	return !b.IsDeletable
}

type ButtonPayload struct {
	Label           string      `json:"label"`
	TextColor       string      `json:"textColor"`
	BackgroundColor string      `json:"backgroundColor"`
	Icon            *string     `json:"icon,omitempty"`
	DisplayMode     DisplayMode `json:"displayMode"`
	Position        int         `json:"position"`
	IsDeletable     bool        `json:"isDeletable"`
}

func (b *ButtonPayload) ToModel() *Button {
	return &Button{
		Label:           b.Label,
		TextColor:       b.TextColor,
		BackgroundColor: b.BackgroundColor,
		Icon:            b.Icon,
		DisplayMode:     b.DisplayMode,
		Position:        b.Position,
		IsDeletable:     b.IsDeletable,
	}
}
