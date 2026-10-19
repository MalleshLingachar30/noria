package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/amxv/noria/internal/model"
)

type statusView struct {
	Bindings int `json:"bindings"`
	Accounts []struct {
		Email          string          `json:"email,omitempty"`
		Alias          string          `json:"alias,omitempty"`
		DisplayName    string          `json:"display_name"`
		ID             string          `json:"id"`
		Provider       string          `json:"provider"`
		Enabled        bool            `json:"enabled"`
		Inflight       int             `json:"inflight"`
		RecentSessions int             `json:"recent_sessions"`
		Snapshot       *model.Snapshot `json:"snapshot"`
		Blockers       []struct {
			Kind  string    `json:"kind"`
			Model string    `json:"model"`
			Until time.Time `json:"until"`
		} `json:"blockers"`
	} `json:"accounts"`
}

func renderStatus(out io.Writer, data []byte, now time.Time) error {
	var view statusView
	if err := json.Unmarshal(data, &view); err != nil {
		return fmt.Errorf("invalid status response: %w", err)
	}
	if len(view.Accounts) == 0 {
		fmt.Fprintln(out, "No accounts registered. Run: noria account add codex")
		return nil
	}
	table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ACCOUNT\tPROVIDER\t5H LEFT / RESET\tWEEK LEFT / RESET\tIN FLIGHT\tSTATE")
	for _, account := range view.Accounts {
		short, week, state := "unknown", "unknown", "quota unknown"
		if account.Snapshot != nil {
			snapshot := account.Snapshot
			if snapshot.Available {
				state = "ready"
			}
			if now.Sub(snapshot.ObservedAt) > 2*time.Minute {
				state = "stale quota"
			}
			if snapshot.AuthInvalid {
				state = "login required"
			}
			for _, window := range snapshot.Windows {
				if window.Model != "" {
					continue
				}
				if window.Duration == 5*time.Hour {
					short = formatWindow(window, now)
				}
				if window.Duration == 7*24*time.Hour {
					week = formatWindow(window, now)
				}
			}
		}
		for _, blocker := range account.Blockers {
			if !blocker.Until.IsZero() && !blocker.Until.After(now) {
				continue
			}
			if blocker.Model != "" {
				state = "model limit"
				continue
			}
			state = blocker.Kind + " blocked"
			if blocker.Kind == "auth" {
				state = "login required"
			}
		}
		if !account.Enabled {
			state = "disabled"
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%d\t%s\n", displayAccount(account.ID, account.Email, account.Alias), account.Provider, short, week, account.Inflight, state)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(out, "\n%d saved session binding(s). Existing sessions keep their account until an applicable limit.\n", view.Bindings)
	fmt.Fprintln(out, "Use noria status --json for model-scoped quotas, timestamps, and routing metadata.")
	return nil
}

func formatWindow(window model.Window, now time.Time) string {
	remaining := "?"
	if window.Used != nil {
		remaining = fmt.Sprintf("%.1f%%", 100-*window.Used)
	}
	reset := "reset unknown"
	if window.ResetsAt != nil {
		duration := window.ResetsAt.Sub(now)
		if duration <= 0 {
			reset = "refresh due"
		} else {
			reset = compactDuration(duration)
		}
	} else if window.Dormant {
		reset = "not started"
	}
	return remaining + " / " + reset
}

func compactDuration(duration time.Duration) string {
	if duration < time.Minute {
		return "<1m"
	}
	minutes := int(duration.Round(time.Minute) / time.Minute)
	days, hours := minutes/1440, minutes%1440/60
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if days == 0 && minutes%60 > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes%60))
	}
	return strings.Join(parts, " ")
}

func displayAccount(id, email, alias string) string {
	if alias != "" {
		return alias
	}
	if email != "" {
		return email
	}
	return id
}
