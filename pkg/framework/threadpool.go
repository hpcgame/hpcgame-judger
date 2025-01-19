package framework

import "sync"

type ThreadPool struct {
	jobs     []func()
	workers  []chan func()
	finCh    chan struct{}
	cancelCh chan struct{}
}

func NewThreadPool(workerNum int) *ThreadPool {
	tp := &ThreadPool{
		jobs:     make([]func(), 0),
		workers:  make([]chan func(), workerNum),
		finCh:    make(chan struct{}),
		cancelCh: make(chan struct{}),
	}
	return tp
}

func (tp *ThreadPool) Add(job func()) {
	tp.jobs = append(tp.jobs, job)
}

func (tp *ThreadPool) Run() {
	wg := &sync.WaitGroup{}

	for i := range tp.workers {
		tp.workers[i] = make(chan func())
		go func(worker chan func()) {
			for job := range worker {
				job()
				wg.Done()
			}
		}(tp.workers[i])
	}

	for i := range tp.jobs {
		wg.Add(1)
		select {
		case <-tp.cancelCh:
			return
		default:
		}

		worker := tp.workers[i%len(tp.workers)]
		worker <- tp.jobs[i]
	}

	for i := range tp.workers {
		close(tp.workers[i])
	}

	wg.Wait()
	close(tp.finCh)
}

func (tp *ThreadPool) Start() {
	go tp.Run()
}

func (tp *ThreadPool) Wait() {
	<-tp.finCh
}

func (tp *ThreadPool) Ch() chan struct{} {
	return tp.finCh
}

func (tp *ThreadPool) Cancel() {
	close(tp.cancelCh)
}
