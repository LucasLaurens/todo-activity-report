package domain

// Optional models a value that may be absent, avoiding nil pointers.
type Optional[T any] struct {
	value   T
	present bool
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, present: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) IsPresent() bool {
	return o.present
}

func (o Optional[T]) Get() (T, bool) {
	return o.value, o.present
}

func (o Optional[T]) OrElse(fallback T) T {
	if o.present {
		return o.value
	}
	return fallback
}
