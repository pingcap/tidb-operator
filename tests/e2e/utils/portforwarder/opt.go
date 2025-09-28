package portforwarder

import "io"

type Options struct {
	Writer io.Writer
}

type Option interface {
	With(*Options)
}

type OptionFunc func(*Options)

func (f OptionFunc) With(opts *Options) {
	f(opts)
}

func WithWriter(w io.Writer) Option {
	return OptionFunc(func(o *Options) {
		o.Writer = w
	})
}
