// Package model defines the shared account and quota vocabulary.
package model

import "time"

type Account struct {
	ID       string   `json:"id"`
	Provider string   `json:"provider"`
	Profile  string   `json:"profile"`
	Enabled  bool     `json:"enabled"`
	Endpoint string   `json:"endpoint,omitempty"`
	Models   []string `json:"models,omitempty"`
}

type Window struct {
	Name      string        `json:"name"`
	LimitID   string        `json:"limit_id,omitempty"`
	Model     string        `json:"model,omitempty"`
	Used      *float64      `json:"used_percent,omitempty"`
	ResetsAt  *time.Time    `json:"resets_at,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Exhausted bool          `json:"exhausted,omitempty"`
	Allowed   *bool         `json:"ordinary_usage_allowed,omitempty"`
	Dormant   bool          `json:"awaiting_first_use,omitempty"`
}

type Snapshot struct {
	AccountID   string    `json:"account_id"`
	ObservedAt  time.Time `json:"observed_at"`
	Available   bool      `json:"available"`
	AuthInvalid bool      `json:"auth_invalid,omitempty"`
	Windows     []Window  `json:"windows,omitempty"`
}

type Credential struct {
	Token     string
	AccountID string
}

type Rejection struct {
	Kind   string // quota, auth, throughput, transient
	Model  string
	Window string
	Until  time.Time
}
