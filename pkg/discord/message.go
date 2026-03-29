package discord

import "time"

// Discord embed color constants.
const (
	ColorGreen  = 3066993  // #2ECC71 — success
	ColorRed    = 15158332 // #E74C3C — error
	ColorYellow = 16776960 // #FFFF00 — warning
	ColorBlue   = 3447003  // #3498DB — info
	ColorGray   = 9807270  // #95A5A6 — neutral
)

// webhookPayload is the Discord webhook request body.
type webhookPayload struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

// Embed represents a Discord embed object.
type Embed struct {
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
	Color       int     `json:"color,omitempty"`
	Timestamp   string  `json:"timestamp,omitempty"`
	Fields      []Field `json:"fields,omitempty"`
	Footer      *Footer `json:"footer,omitempty"`
	Author      *Author `json:"author,omitempty"`
	Thumbnail   *Image  `json:"thumbnail,omitempty"`
}

// Field represents an embed field.
type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// Footer represents the embed footer.
type Footer struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// Author represents the embed author.
type Author struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// Image represents an embed image/thumbnail.
type Image struct {
	URL string `json:"url"`
}

// NewEmbed creates an Embed with title, color, and current timestamp.
func NewEmbed(title string, color int) Embed {
	return Embed{
		Title:     title,
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// WithField adds a field to the embed and returns it.
func (e Embed) WithField(name, value string, inline bool) Embed {
	e.Fields = append(e.Fields, Field{Name: name, Value: value, Inline: inline})
	return e
}

// WithFooter sets the footer text.
func (e Embed) WithFooter(text string) Embed {
	e.Footer = &Footer{Text: text}
	return e
}

// WithAuthor sets the author.
func (e Embed) WithAuthor(name, url, iconURL string) Embed {
	e.Author = &Author{Name: name, URL: url, IconURL: iconURL}
	return e
}

// WithDescription sets the description.
func (e Embed) WithDescription(desc string) Embed {
	e.Description = desc
	return e
}

// WithURL sets the embed URL.
func (e Embed) WithURL(url string) Embed {
	e.URL = url
	return e
}
