package pool

type Pool[T any] struct {
	items     chan T
	newFunc   func() T
	reuseFunc func(T) bool
}

type Option[T any] func(*Pool[T])

// optional param, adds condition to decide if pool should be reused
func WithReuseCheck[T any](reuseFunc func(T) bool) Option[T] {
	return func(p *Pool[T]) {
		p.reuseFunc = reuseFunc
	}
}

func New[T any](size int, newFunc func() T, opts ...Option[T]) *Pool[T] {
	pool := &Pool[T]{
		items:     make(chan T, size),
		newFunc:   newFunc,
		reuseFunc: func(T) bool { return true },
	}

	for _, opt := range opts {
		opt(pool)
	}

	return pool
}

func (p *Pool[T]) Get() T {
	select {
	case item := <-p.items:
		return item
	default:
		return p.newFunc()
	}
}

func (p *Pool[T]) Put(item T) bool {
	// drop unreusable items if reuseFunc provided
	if p.reuseFunc != nil && !p.reuseFunc(item) {
		return false
	}

	select {
	case p.items <- item:
		return true
	default:
		return false
	}
}
