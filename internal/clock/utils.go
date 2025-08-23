package clock

import (
	"fmt"
	"strings"
	"time"
)

// TimeFormatter handles time formatting utilities
type TimeFormatter struct{}

// NewTimeFormatter creates a new time formatter
func NewTimeFormatter() *TimeFormatter {
	return &TimeFormatter{}
}

// FormatDuration formats a duration as MM:SS
func (tf *TimeFormatter) FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}

	// Round to nearest second
	d = d.Round(time.Second)

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// FormatDurationLong formats a duration with hours if needed
func (tf *TimeFormatter) FormatDurationLong(d time.Duration) string {
	if d <= 0 {
		return "0 minutes"
	}

	// Round to nearest minute
	d = d.Round(time.Minute)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%d hours %d minutes", hours, minutes)
		}
		return fmt.Sprintf("%d hours", hours)
	}

	return fmt.Sprintf("%d minutes", minutes)
}

// FormatTimeRemaining formats remaining time with context
func (tf *TimeFormatter) FormatTimeRemaining(remaining time.Duration) string {
	if remaining <= 0 {
		return "Time's up!"
	}

	return fmt.Sprintf("%s remaining", tf.FormatDuration(remaining))
}

// ParseDurationString parses a duration string (e.g., "25m", "5m", "1h30m")
func (tf *TimeFormatter) ParseDurationString(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// FormatDurationString formats a duration as a string (e.g., "25m", "1h30m")
func (tf *TimeFormatter) FormatDurationString(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	// Round to nearest second
	d = d.Round(time.Second)

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	if minutes > 0 {
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%ds", seconds)
}

// ClockUtils provides general clock utility functions
type ClockUtils struct{}

// NewClockUtils creates a new clock utilities instance
func NewClockUtils() *ClockUtils {
	return &ClockUtils{}
}

// IsValidDuration checks if a duration is valid for pomodoro sessions
func (cu *ClockUtils) IsValidDuration(d time.Duration) bool {
	// Minimum 1 minute, maximum 4 hours
	return d >= time.Minute && d <= 4*time.Hour
}

// GetRecommendedDurations returns recommended pomodoro durations
func (cu *ClockUtils) GetRecommendedDurations() map[string]time.Duration {
	return map[string]time.Duration{
		"classic":     25 * time.Minute,
		"short":       15 * time.Minute,
		"long":        45 * time.Minute,
		"micro":       5 * time.Minute,
		"short_break": 5 * time.Minute,
		"long_break":  15 * time.Minute,
		"micro_break": 1 * time.Minute,
	}
}

// CalculateSessionProgress calculates the progress percentage of a session
func (cu *ClockUtils) CalculateSessionProgress(elapsed, total time.Duration) float64 {
	if total <= 0 {
		return 0.0
	}

	progress := float64(elapsed) / float64(total) * 100.0
	if progress > 100.0 {
		progress = 100.0
	}

	return progress
}

// EstimateCompletionTime estimates when a session will complete
func (cu *ClockUtils) EstimateCompletionTime(remaining time.Duration) time.Time {
	return time.Now().Add(remaining)
}

// IsSessionComplete checks if a session is complete based on elapsed time
func (cu *ClockUtils) IsSessionComplete(elapsed, total time.Duration) bool {
	return elapsed >= total
}

// GetTimeUntilNextSession calculates time until the next session starts
func (cu *ClockUtils) GetTimeUntilNextSession(currentTime, nextSessionTime time.Time) time.Duration {
	if nextSessionTime.Before(currentTime) {
		return 0
	}
	return nextSessionTime.Sub(currentTime)
}

// FormatSessionInfo formats session information for display
func (cu *ClockUtils) FormatSessionInfo(state ClockState, sessionNum, totalSessions int, duration time.Duration) string {
	formatter := NewTimeFormatter()

	switch state {
	case StateWorking:
		return fmt.Sprintf("Work Session %d/%d (%s)", sessionNum, totalSessions, formatter.FormatDuration(duration))
	case StateShortBreak:
		return fmt.Sprintf("Short Break %d/%d (%s)", sessionNum, totalSessions, formatter.FormatDuration(duration))
	case StateLongBreak:
		return fmt.Sprintf("Long Break %d/%d (%s)", sessionNum, totalSessions, formatter.FormatDuration(duration))
	case StatePaused:
		return "Session Paused"
	case StateIdle:
		return "Ready to Start"
	default:
		return "Unknown State"
	}
}

// ValidateSchedule validates a pomodoro schedule
func (cu *ClockUtils) ValidateSchedule(schedule []ClockState) error {
	if len(schedule) == 0 {
		return fmt.Errorf("schedule cannot be empty")
	}

	for i, state := range schedule {
		if state != StateWorking && state != StateShortBreak && state != StateLongBreak {
			return fmt.Errorf("invalid state at position %d: %s", i, state)
		}
	}

	return nil
}

// GetScheduleSummary returns a summary of a schedule
func (cu *ClockUtils) GetScheduleSummary(schedule []ClockState) map[ClockState]int {
	summary := make(map[ClockState]int)

	for _, state := range schedule {
		summary[state]++
	}

	return summary
}

func ParseScheduling(schedulingString string) ([]ClockState, error) {
	// remove all whitespace, newlines and tabs
	schedulingString = strings.ReplaceAll(schedulingString, " ", "")
	schedulingString = strings.ReplaceAll(schedulingString, "\n", "")
	schedulingString = strings.ReplaceAll(schedulingString, "\t", "")

	tokens := strings.Split(schedulingString, "-")
	schedule := make([]ClockState, 0, len(tokens))
	for _, token := range tokens {
		// check if token is a valid ClockState
		if _, ok := ClockStateMap[token]; ok {
			schedule = append(schedule, ClockState(token))
		} else {
			return nil, fmt.Errorf("invalid token: %s, valid tokens are: %v", token, ClockStateMap)
		}
	}
	return schedule, nil
}

// FormatScheduling converts a slice of ClockState to a string format joined by "-"
func FormatScheduling(schedule []ClockState) string {
	if len(schedule) == 0 {
		return ""
	}

	// Convert ClockState values to their string representations
	tokens := make([]string, len(schedule))
	for i, state := range schedule {
		switch state {
		case StateWorking:
			tokens[i] = string(StateWorking)
		case StateShortBreak:
			tokens[i] = string(StateShortBreak)
		case StateLongBreak:
			tokens[i] = string(StateLongBreak)
		default:
			tokens[i] = string(state)
		}
	}

	return strings.Join(tokens, "-")
}
