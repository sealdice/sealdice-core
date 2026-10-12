package sender

import "sync"

type Queue struct {
	mu     sync.Mutex
	queues map[string]*keyQueue
	wg     sync.WaitGroup
	closed bool
}

type keyQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	jobs   []func()
	closed bool
}

func New() *Queue {
	return &Queue{queues: make(map[string]*keyQueue)}
}

func (q *Queue) Enqueue(key string, job func()) {
	if job == nil {
		return
	}

	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	kq, ok := q.queues[key]
	if !ok {
		kq = &keyQueue{}
		kq.cond = sync.NewCond(&kq.mu)
		q.queues[key] = kq
		q.wg.Add(1)
		go kq.run(&q.wg)
	}
	q.mu.Unlock()

	kq.push(job)
}

func (q *Queue) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	for _, kq := range q.queues {
		kq.close()
	}
	q.queues = nil
	q.mu.Unlock()

	q.wg.Wait()
}

func (kq *keyQueue) push(job func()) {
	kq.mu.Lock()
	if kq.closed {
		kq.mu.Unlock()
		return
	}
	kq.jobs = append(kq.jobs, job)
	kq.cond.Signal()
	kq.mu.Unlock()
}

func (kq *keyQueue) run(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		kq.mu.Lock()
		for len(kq.jobs) == 0 && !kq.closed {
			kq.cond.Wait()
		}
		if len(kq.jobs) == 0 {
			kq.mu.Unlock()
			return
		}
		job := kq.jobs[0]
		kq.jobs[0] = nil
		kq.jobs = kq.jobs[1:]
		if len(kq.jobs) == 0 {
			kq.jobs = nil
		}
		kq.mu.Unlock()

		job()
	}
}

func (kq *keyQueue) close() {
	kq.mu.Lock()
	kq.closed = true
	kq.cond.Broadcast()
	kq.mu.Unlock()
}
