// Package xredis 提供一个带缓冲队列与工作池的 Redis 异步/同步操作模块。

package xredis

import (
    "context"
    "math"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

type Config struct {
    Addr      string
    Password  string
    DB        int
    Workers   int
    QueueSize int
}

type DB struct {
    rdb *redis.Client
    cfg Config

    jobs chan job
    wg   sync.WaitGroup

    ctx    context.Context
    cancel context.CancelFunc
}

type job struct {
    fn    func(redis.Cmdable) error
    tries int
}

func Open(parent context.Context, cfg Config) (*DB, error) {
    if cfg.Workers <= 0 {
        cfg.Workers = 2
    }
    if cfg.QueueSize <= 0 {
        cfg.QueueSize = 1000
    }

    rdb := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    if err := rdb.Ping(parent).Err(); err != nil {
        return nil, err
    }

    ctx, cancel := context.WithCancel(parent)
    db := &DB{
        rdb:    rdb,
        cfg:    cfg,
        jobs:   make(chan job, cfg.QueueSize),
        ctx:    ctx,
        cancel: cancel,
    }

    for i := 0; i < cfg.Workers; i++ {
        db.wg.Add(1)
        go db.worker()
    }

    return db, nil
}

func (db *DB) worker() {
    defer db.wg.Done()
    for {
        select {
        case <-db.ctx.Done():
            return
        case j, ok := <-db.jobs:
            if !ok {
                return
            }
            _ = db.execWithRetry(j)
        }
    }
}

func (db *DB) execWithRetry(j job) error {
    if err := j.fn(db.rdb); err != nil {
        if j.tries < 5 {
            wait := time.Duration(math.Pow(2, float64(j.tries))) * 100 * time.Millisecond
            timer := time.NewTimer(wait)
            select {
            case <-db.ctx.Done():
                timer.Stop()
                return err
            case <-timer.C:
                j.tries++
                select {
                case db.jobs <- j:
                default:
                }
            }
        }
        return err
    }
    return nil
}

// Enqueue 异步执行一个 redis 命令
func (db *DB) Enqueue(fn func(redis.Cmdable) error) {
    db.jobs <- job{fn: fn}
}

// ExecSync 同步执行一个 redis 命令（立即执行，不入队）
func (db *DB) ExecSync(ctx context.Context, fn func(redis.Cmdable) error) error {
    return fn(db.rdb) // 直接用 db.rdb 就行
}


func (db *DB) Close() error {
    db.cancel()
    close(db.jobs)
    db.wg.Wait()
    return db.rdb.Close()
}
