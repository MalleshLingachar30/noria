//go:build !darwin

package app

// The daemon owns background updates on Linux and Windows; macOS uses
// a launchd job so checks also run when neither app nor router is open.
func syncUpdateAgent(string, string) error { return nil }
