package ai

import "context"

type Priority int

const (
	PriorityHigh Priority = iota
	PriorityLow
)

type Options struct {
	Fast        bool
	MaxTokens   int
	Priority    Priority
	System      string
	Temperature *float64
}

type Option func(*Options)

func WithFastModel() Option {
	return func(o *Options) { o.Fast = true }
}

func WithMaxTokens(n int) Option {
	return func(o *Options) { o.MaxTokens = n }
}

func WithLowPriority() Option {
	return func(o *Options) { o.Priority = PriorityLow }
}

// WithSystem sends fixed instructions as a system message, so a local model
// server can reuse its cache for the unchanging start of the prompt.
func WithSystem(instructions string) Option {
	return func(o *Options) { o.System = instructions }
}

func WithTemperature(t float64) Option {
	return func(o *Options) { o.Temperature = &t }
}

func applyOptions(opts []Option) Options {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

type lowPriorityKey struct{}

// LowPriorityContext marks embedding calls made with ctx as background work
// that waits while a member is waiting for the language model.
func LowPriorityContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, lowPriorityKey{}, true)
}

func isLowPriority(ctx context.Context) bool {
	v, _ := ctx.Value(lowPriorityKey{}).(bool)
	return v
}

func ResolveOptions(opts ...Option) Options {
	return applyOptions(opts)
}
