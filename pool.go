package websocket

// Pool of goroutines that have a max concurrency.
type pool struct {
	work chan func()
	sem  chan struct{}
}

// Create a new pool.
func newPool(concurrency int) *pool {
	return &pool{
		work: make(chan func()),
		sem:  make(chan struct{}, concurrency),
	}
}

// Schedule a task on the pool.
func (p *pool) schedule(task func()) {
	select {
	case p.work <- task:
	case p.sem <- struct{}{}:
		go p.worker(task)
	}
}

// Perform the scheduled work.
func (p *pool) worker(task func()) {
	defer func() { <-p.sem }()
	for {
		task()
		task = <-p.work
	}
}
