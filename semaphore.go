package main

// Semaphore is a very simple syncronization semaphore.
type Semaphore struct {
	channel chan struct{}
}

// NewSemaphore initializes a Semmaphore with a given capacity.
func NewSemaphore(cap int) *Semaphore {
	return &Semaphore{
		channel: make(chan struct{}, cap),
	}
}

// Len returns current number of taken resources in the semaphore.
func (sm *Semaphore) Len() int {
	return len(sm.channel)
}

// Cap returns the total capacity of the semaphore.
func (sm *Semaphore) Cap() int {
	return cap(sm.channel)
}

// Acquire attempts to acquire the semaphore for a single resource.
//
// If semaphore is currently full, will block until one becomes avaiable.
func (sm *Semaphore) Acquire() {
	sm.channel <- struct{}{}
}

// Done marks a job as done and frees a single resource.
func (sm *Semaphore) Done() {
	<-sm.channel
}
