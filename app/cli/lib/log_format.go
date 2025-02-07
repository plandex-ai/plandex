package lib

import (
	"regexp"
	"strings"
	"time"
)

// Regular expressions for parsing log entries
var (
	// Match the update ID and timestamp
	updateRegex = regexp.MustCompile(`📝 Update ([a-f0-9]+)\[0;22m \| \[36m(.*?)\[0m`)
	
	// Match message number and type
	messageRegex = regexp.MustCompile(`Message #(\d+) \| (.+?) \|`)
	
	// Match coin count
	coinRegex = regexp.MustCompile(`(\d+) 🪙`)
	
	// Match context load summary
	contextRegex = regexp.MustCompile(`Loaded (\d+) .+ into context`)
)

// LogEntry represents a parsed log entry
type LogEntry struct {
	ID        string
	Timestamp string
	Type      string
	Message   string
}

// ParseLogEntry parses a raw log entry string into a structured LogEntry
func ParseLogEntry(raw string) LogEntry {
	lines := strings.Split(raw, "\n")
	entry := LogEntry{}
	
	// Parse first line for update ID and timestamp
	if matches := updateRegex.FindStringSubmatch(lines[0]); len(matches) >= 3 {
		entry.ID = matches[1]
		entry.Timestamp = parseTimestamp(matches[2])
	}
	
	// Parse second line for message type and details
	if len(lines) > 1 {
		if matches := messageRegex.FindStringSubmatch(lines[1]); len(matches) >= 3 {
			entry.Type = cleanType(matches[2])
			entry.Message = lines[1]
		} else if matches := contextRegex.FindStringSubmatch(lines[1]); len(matches) >= 2 {
			entry.Type = "Context Load"
			entry.Message = lines[1]
		}
	}
	
	return entry
}

// FormatCompactSummary creates a compact one-line summary of the log entry
func FormatCompactSummary(entry LogEntry) string {
	var summary strings.Builder
	
	// Add timestamp
	summary.WriteString(entry.Timestamp)
	summary.WriteString(" | ")
	
	// Add type indicator and summary based on type
	switch {
	case strings.Contains(entry.Type, "User prompt"):
		summary.WriteString("💬 User: ")
		msg := extractFirstLine(entry.Message)
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}
		summary.WriteString(msg)
		
	case strings.Contains(entry.Type, "Plandex reply"):
		summary.WriteString("🤖 AI: ")
		if coins := coinRegex.FindStringSubmatch(entry.Message); len(coins) >= 2 {
			summary.WriteString(coins[1] + "🪙")
		}
		
	case strings.Contains(entry.Type, "Context Load"):
		summary.WriteString("📚 ")
		if matches := contextRegex.FindStringSubmatch(entry.Message); len(matches) >= 2 {
			summary.WriteString("Loaded " + matches[1] + " items")
		}
		
	case strings.Contains(entry.Type, "Build"):
		summary.WriteString("🏗️ Build changes")
		
	default:
		summary.WriteString(entry.Type)
	}
	
	return summary.String()
}

// Helper functions

func parseTimestamp(ts string) string {
	// Convert timestamp to a consistent format
	ts = strings.TrimSpace(ts)
	if ts == "Today" {
		return time.Now().Format("15:04:05")
	}
	if ts == "Yesterday" {
		return "Yesterday"
	}
	return ts
}

func cleanType(t string) string {
	// Remove ANSI color codes and clean up type string
	t = strings.TrimSpace(t)
	return strings.ReplaceAll(t, "[0m", "")
}

func extractFirstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx != -1 {
		return s[:idx]
	}
	return s
}
