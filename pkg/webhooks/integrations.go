package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text        string              `json:"text,omitempty"`
	Blocks      []SlackBlock        `json:"blocks,omitempty"`
	Attachments []SlackAttachment   `json:"attachments,omitempty"`
}

// SlackBlock represents a Slack block
type SlackBlock struct {
	Type string          `json:"type"`
	Text *SlackBlockText `json:"text,omitempty"`
}

// SlackBlockText represents text in a Slack block
type SlackBlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackAttachment represents a Slack attachment
type SlackAttachment struct {
	Color  string `json:"color,omitempty"`
	Title  string `json:"title,omitempty"`
	Text   string `json:"text,omitempty"`
	Fields []SlackField `json:"fields,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// TeamsMessage represents a Microsoft Teams webhook message
type TeamsMessage struct {
	Type       string          `json:"@type"`
	Context    string          `json:"@context"`
	Summary    string          `json:"summary,omitempty"`
	Title      string          `json:"title,omitempty"`
	Text       string          `json:"text,omitempty"`
	ThemeColor string          `json:"themeColor,omitempty"`
	Sections   []TeamsSection  `json:"sections,omitempty"`
}

// TeamsSection represents a section in a Teams message
type TeamsSection struct {
	ActivityTitle    string       `json:"activityTitle,omitempty"`
	ActivitySubtitle string       `json:"activitySubtitle,omitempty"`
	ActivityImage    string       `json:"activityImage,omitempty"`
	Facts            []TeamsFact  `json:"facts,omitempty"`
	Text             string       `json:"text,omitempty"`
}

// TeamsFact represents a fact in a Teams section
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FormatSlackMessage formats an event as a Slack message
func FormatSlackMessage(event *Event) SlackMessage {
	color := getEventColor(event.Type)
	title := getEventTitle(event.Type)

	fields := []SlackField{
		{Title: "Event Type", Value: string(event.Type), Short: true},
		{Title: "Event ID", Value: event.ID, Short: true},
		{Title: "Timestamp", Value: event.Timestamp.Format("2006-01-02 15:04:05"), Short: true},
	}

	// Add event-specific fields
	if module, ok := event.Data["module"].(string); ok {
		fields = append(fields, SlackField{Title: "Module", Value: module, Short: true})
	}
	if version, ok := event.Data["version"].(string); ok {
		fields = append(fields, SlackField{Title: "Version", Value: version, Short: true})
	}
	if message, ok := event.Data["message"].(string); ok {
		fields = append(fields, SlackField{Title: "Message", Value: message, Short: false})
	}

	return SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color:  color,
				Title:  title,
				Fields: fields,
			},
		},
	}
}

// FormatTeamsMessage formats an event as a Microsoft Teams message
func FormatTeamsMessage(event *Event) TeamsMessage {
	themeColor := getEventThemeColor(event.Type)
	title := getEventTitle(event.Type)

	facts := []TeamsFact{
		{Name: "Event Type", Value: string(event.Type)},
		{Name: "Event ID", Value: event.ID},
		{Name: "Timestamp", Value: event.Timestamp.Format("2006-01-02 15:04:05")},
	}

	// Add event-specific facts
	if module, ok := event.Data["module"].(string); ok {
		facts = append(facts, TeamsFact{Name: "Module", Value: module})
	}
	if version, ok := event.Data["version"].(string); ok {
		facts = append(facts, TeamsFact{Name: "Version", Value: version})
	}

	var text string
	if message, ok := event.Data["message"].(string); ok {
		text = message
	}

	return TeamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		Summary:    title,
		Title:      title,
		ThemeColor: themeColor,
		Sections: []TeamsSection{
			{
				Facts: facts,
				Text:  text,
			},
		},
	}
}

// SendSlackNotification sends a notification to a Slack webhook
func SendSlackNotification(ctx context.Context, webhookURL string, event *Event) error {
	message := FormatSlackMessage(event)
	return sendJSON(ctx, webhookURL, message)
}

// SendTeamsNotification sends a notification to a Microsoft Teams webhook
func SendTeamsNotification(ctx context.Context, webhookURL string, event *Event) error {
	message := FormatTeamsMessage(event)
	return sendJSON(ctx, webhookURL, message)
}

// sendJSON sends a JSON payload to a URL
func sendJSON(ctx context.Context, url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// getEventColor returns the Slack color for an event type
func getEventColor(eventType EventType) string {
	switch eventType {
	case EventModuleCreated, EventVersionCreated, EventCompilationComplete:
		return "good" // Green
	case EventCompilationFailed, EventBreakingChange, EventValidationFailed:
		return "danger" // Red
	case EventCompilationStarted:
		return "warning" // Yellow
	default:
		return "#439FE0" // Blue
	}
}

// getEventThemeColor returns the Teams theme color for an event type
func getEventThemeColor(eventType EventType) string {
	switch eventType {
	case EventModuleCreated, EventVersionCreated, EventCompilationComplete:
		return "28a745" // Green
	case EventCompilationFailed, EventBreakingChange, EventValidationFailed:
		return "dc3545" // Red
	case EventCompilationStarted:
		return "ffc107" // Yellow
	default:
		return "007bff" // Blue
	}
}

// getEventTitle returns a human-readable title for an event type
func getEventTitle(eventType EventType) string {
	switch eventType {
	case EventModuleCreated:
		return "Module Created"
	case EventModuleUpdated:
		return "Module Updated"
	case EventModuleDeleted:
		return "Module Deleted"
	case EventVersionCreated:
		return "Version Created"
	case EventVersionDeleted:
		return "Version Deleted"
	case EventCompilationStarted:
		return "Compilation Started"
	case EventCompilationComplete:
		return "Compilation Complete"
	case EventCompilationFailed:
		return "Compilation Failed"
	case EventBreakingChange:
		return "Breaking Change Detected"
	case EventValidationFailed:
		return "Validation Failed"
	default:
		return string(eventType)
	}
}
