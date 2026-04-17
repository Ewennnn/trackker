package formatter

import (
	"bytes"
	"djtracker/internal/model"
	"html/template"
	"regexp"
)

var whitespacesRegex = regexp.MustCompile(`[\r\n]+`)

type HtmlFormatter struct {
	tmpl *template.Template
}

func (p *HtmlFormatter) Format(track *model.Track) (string, error) {
	var buf bytes.Buffer
	err := p.tmpl.Execute(&buf, track)
	if err != nil {
		return "", err
	}

	flattened := whitespacesRegex.ReplaceAllString(buf.String(), "")
	return flattened, err
}
