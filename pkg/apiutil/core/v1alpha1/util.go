package coreutil

func setIfChanged[T comparable](dst *T, src T) bool {
	if *dst != src {
		*dst = src
		return true
	}

	return false
}

func newAndSetIfChanged[T comparable](dst **T, src T) bool {
	if *dst == nil {
		zero := new(T)
		if *zero == src {
			return false
		}
		*dst = zero
	}
	return setIfChanged(*dst, src)
}
