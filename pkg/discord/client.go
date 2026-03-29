// pkg/discord — Discord webhook client สำหรับส่ง notification
//
// รองรับ embed message แบบ rich format พร้อม OTel tracing
//
// วิธีใช้:
//
//	client := discord.NewClient(webhookURL)
//	embed := discord.Embed{
//	    Title: "✅ Deployed",
//	    Color: discord.ColorGreen,
//	    Fields: []discord.Field{{Name: "Branch", Value: "main", Inline: true}},
//	}
//	err := client.SendEmbed(ctx, embed)
package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
)

// Client sends messages to a Discord webhook URL.
type Client struct {
	webhookURL string
	httpClient *http.Client
}

// NewClient creates a Discord webhook client.
func NewClient(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// SendEmbed sends a single embed message to Discord.
func (c *Client) SendEmbed(ctx context.Context, embed Embed) error {
	payload := webhookPayload{Embeds: []Embed{embed}}
	return c.send(ctx, payload)
}

// SendEmbeds sends multiple embeds in one message.
func (c *Client) SendEmbeds(ctx context.Context, embeds []Embed) error {
	payload := webhookPayload{Embeds: embeds}
	return c.send(ctx, payload)
}

// SendText sends a plain text message.
func (c *Client) SendText(ctx context.Context, content string) error {
	payload := webhookPayload{Content: content}
	return c.send(ctx, payload)
}

func (c *Client) send(ctx context.Context, payload webhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("discord: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
