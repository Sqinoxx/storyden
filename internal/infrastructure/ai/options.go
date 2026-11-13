package ai

type Priority int

const (
	PriorityHigh Priority = iota
	PriorityLow
)

type Options struct {
	Fast      bool
	MaxTokens int
	Priority  Priority
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

func applyOptions(opts []Option) Options {
	var o Options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
