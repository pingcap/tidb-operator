package client

type ApplyOptions struct {
	// Immutable defines fields which is immutable
	// It's only for some fields which cannot be changed but actually maybe changed.
	// For example,
	// We change storage class ref in instance CR to modify volumes if feature VolumeAttributesClass is not enabled, but it's immutable in PVC. When we enable VolumeAttributesClass, we have to skip storage class change when apply.
	// NOTE; now slice/array is not supported
	Immutable [][]string
}

type ApplyOption interface {
	With(opts *ApplyOptions)
}

type ApplyOptionFunc func(opts *ApplyOptions)

func (f ApplyOptionFunc) With(opts *ApplyOptions) {
	f(opts)
}

func Immutable(fields ...string) ApplyOption {
	return ApplyOptionFunc(func(opts *ApplyOptions) {
		if len(fields) == 0 {
			return
		}
		opts.Immutable = append(opts.Immutable, fields)
	})
}
