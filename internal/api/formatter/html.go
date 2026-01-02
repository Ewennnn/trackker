package formatter

import (
	"bytes"
	"djtracker/internal/model"
	"djtracker/internal/utils"
	"html/template"
	"regexp"
)

var whitespacesRegex = regexp.MustCompile(`[\r\n]+`)

type trackView struct {
	ID     int64
	Artist string
	Title  string
	Cover  string
}

func newTrackView(t *model.Track) *trackView {
	return &trackView{
		ID:     t.ID,
		Artist: utils.SafePointer(t.Artist),
		Title:  t.Name,
		Cover:  utils.SafePointer(t.Cover),
	}
}

type HtmlFormatter struct {
	tmpl *template.Template
}

func (p *HtmlFormatter) Format(track *model.Track) (string, error) {
	view := newTrackView(track)
	var buf bytes.Buffer
	err := p.tmpl.Execute(&buf, view)
	if err != nil {
		return "", err
	}

	flattened := whitespacesRegex.ReplaceAllString(buf.String(), "")
	return flattened, err
}
