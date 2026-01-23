package log_parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Log struct {
	Outcome    string `json:"outcome"`
	ScriptName string `json:"scriptName"`
	Logs       []struct {
		Message   []string `json:"message"`
		Level     string   `json:"level"`
		Timestamp int64    `json:"timestamp"`
	} `json:"logs"`
	Event struct {
		Request struct {
			URL    string `json:"url"`
			Method string `json:"method"`
		} `json:"request"`
		Response struct {
			Status int `json:"status"`
		} `json:"response"`
	} `json:"event"`
}

type ParsedLog struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Method  string    `json:"method"`
	URL     string    `json:"url"`
	Status  int       `json:"status"`
	Message string    `json:"message"`
}

func Parse(data []byte) ([]ParsedLog, error) {
	var raw Log
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	// If no console.logs occurred, we return an empty slice (or skip)
	if len(raw.Logs) == 0 {
		return nil, nil
	}

	parsedEntries := make([]ParsedLog, 0, len(raw.Logs))

	for _, l := range raw.Logs {
		// Convert slice of interfaces to a single readable string
		msgParts := make([]string, len(l.Message))
		for i, part := range l.Message {
			msgParts[i] = fmt.Sprint(part)
		}

		entry := ParsedLog{
			Time:    time.UnixMilli(l.Timestamp),
			Level:   strings.ToUpper(l.Level),
			Method:  raw.Event.Request.Method,
			URL:     raw.Event.Request.URL,
			Status:  raw.Event.Response.Status,
			Message: strings.Join(msgParts, " "),
		}
		parsedEntries = append(parsedEntries, entry)
	}

	return parsedEntries, nil
}