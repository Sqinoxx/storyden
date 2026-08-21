package ai

import "github.com/Southclaws/fault"

// Sentinels for the failure modes callers need to branch on. Batch jobs in
// particular must tell "wait and try again" apart from "this will never work",
// since a quota exhausted for the day would otherwise burn through every
// remaining item repeating the same failure.
var (
	// ErrRateLimited means the provider refused the request for quota reasons
	// and retrying within this run will not help.
	ErrRateLimited = fault.New("ai: rate limited")

	// ErrProviderUnavailable covers transient server-side failures.
	ErrProviderUnavailable = fault.New("ai: provider unavailable")

	// ErrStructuredUnsupported means the configured provider cannot produce
	// structured JSON output, typically because none is configured at all.
	ErrStructuredUnsupported = fault.New("ai: provider does not support structured output")

	// ErrNotConfigured means the provider was selected but is missing its
	// credentials.
	ErrNotConfigured = fault.New("ai: provider is not configured")
)
