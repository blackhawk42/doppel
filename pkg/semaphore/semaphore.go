package semaphore

// Semaphore is a very simple semaphore
type Semaphore struct {
	c chan struct{}
}

// NewSemaphore creates a new semaphore with a given amount of permits before
// blocking.
func NewSemaphore(permits int) *Semaphore {
	return &Semaphore{
		c: make(chan struct{}, permits),
	}
}

// Acquire acquires one of the permits of the semaphore, blocking if none available.
func (s *Semaphore) Acquire() {
	s.c <- struct{}{}
}

// Release releases one of the semaphore's permits, making it available.
func (s *Semaphore) Release() {
	<-s.c
}
