package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"utils/concurrency"
)

func main() {
	// ---- SafeGo 用法 ----
	var wg sync.WaitGroup
	concurrency.SafeGo(func() {
		fmt.Println("安全 goroutine 运行中")
		panic("测试 panic") // 不会崩溃
	}, &wg)
	wg.Wait()

	// ---- WorkerPool 用法 ----
	pool := concurrency.NewWorkerPool(3, 10)
	for i := 0; i < 5; i++ {
		jobID := i
		pool.Submit(func() {
			fmt.Printf("执行任务 %d\n", jobID)
		})
	}
	pool.Close()

	// ---- Semaphore 用法 ----
	sem := concurrency.NewSemaphore(2) // 最大并发 2
	for i := 0; i < 5; i++ {
		idx := i
		concurrency.SafeGo(func() {
			sem.WithSemaphore(func() {
				fmt.Printf("任务 %d 开始\n", idx)
				time.Sleep(1 * time.Second)
				fmt.Printf("任务 %d 完成\n", idx)
			})
		}, nil)
	}
	time.Sleep(3 * time.Second)

	// ---- GoWithContext 用法 ----
	ctx, cancel := context.WithCancel(context.Background())
	concurrency.GoWithContext(ctx, func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("goroutine 被取消")
				return
			default:
				fmt.Println("goroutine 正在运行...")
				time.Sleep(500 * time.Millisecond)
			}
		}
	})
	time.Sleep(2 * time.Second)
	cancel()
	time.Sleep(1 * time.Second)
}
