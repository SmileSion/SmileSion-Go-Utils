// concurrency.go
package concurrency

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
)

// =============================
// SafeGo：安全启动 goroutine
// =============================
// 自动 recover，避免 goroutine panic 导致整个程序崩溃
func SafeGo(fn func(), wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeGo] panic recovered: %v\n%s", r, debug.Stack())
			}
			if wg != nil {
				wg.Done()
			}
		}()
		fn()
	}()
}

// =============================
// Worker Pool：固定数量的 worker
// =============================
type WorkerPool struct {
	wg      sync.WaitGroup
	jobChan chan func()
}

// NewWorkerPool 创建一个 worker 池
func NewWorkerPool(workerCount int, jobQueueSize int) *WorkerPool {
	pool := &WorkerPool{
		jobChan: make(chan func(), jobQueueSize),
	}
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go func(id int) {
			defer pool.wg.Done()
			for job := range pool.jobChan {
				SafeGo(job, nil) // 每个任务用 SafeGo 包裹，防止 panic
			}
		}(i)
	}
	return pool
}

// Submit 提交任务
func (p *WorkerPool) Submit(job func()) {
	p.jobChan <- job
}

// Close 关闭池子并等待所有任务完成
func (p *WorkerPool) Close() {
	close(p.jobChan)
	p.wg.Wait()
}

// =============================
// Semaphore：并发限制
// =============================
type Semaphore struct {
	tokens chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{tokens: make(chan struct{}, limit)}
}

func (s *Semaphore) Acquire() {
	s.tokens <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.tokens
}

// WithSemaphore 执行函数时限制最大并发
func (s *Semaphore) WithSemaphore(fn func()) {
	s.Acquire()
	defer s.Release()
	fn()
}

// =============================
// Context Wrapper：带 cancel 的 goroutine
// =============================
func GoWithContext(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[GoWithContext] panic recovered: %v\n%s", r, debug.Stack())
			}
		}()
		fn(ctx)
	}()
}
