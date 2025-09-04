// Package xmysql 提供一个带缓冲队列与工作池的 MySQL 异步/同步读写模块。


package xmysql

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "math"
    "sync"
    "time"

    _ "github.com/go-sql-driver/mysql"
)

type Config struct {
    DSN        string        // "user:pass@tcp(host:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
    Workers    int
    QueueSize  int
    MaxOpen    int
    MaxIdle    int
    MaxLife    time.Duration
}

type Option func(*openOptions)

type openOptions struct {
    migrations []string
}

func WithMigrations(sqls []string) Option {
    return func(o *openOptions) { o.migrations = sqls }
}

type DB struct {
    sqldb *sql.DB
    cfg   Config

    jobs chan job
    wg   sync.WaitGroup

    ctx    context.Context
    cancel context.CancelFunc
}

type job struct {
    query string
    args  []any
    tries int
}

func Open(parent context.Context, cfg Config, opts ...Option) (*DB, error) {
    if cfg.DSN == "" {
        return nil, errors.New("DSN required")
    }
    if cfg.Workers <= 0 {
        cfg.Workers = 4
    }
    if cfg.QueueSize <= 0 {
        cfg.QueueSize = 1000
    }
    if cfg.MaxOpen <= 0 {
        cfg.MaxOpen = 20
    }
    if cfg.MaxIdle <= 0 {
        cfg.MaxIdle = cfg.Workers
    }

    o := &openOptions{}
    for _, f := range opts {
        f(o)
    }

    sqldb, err := sql.Open("mysql", cfg.DSN)
    if err != nil {
        return nil, err
    }
    sqldb.SetMaxOpenConns(cfg.MaxOpen)
    sqldb.SetMaxIdleConns(cfg.MaxIdle)
    sqldb.SetConnMaxLifetime(cfg.MaxLife)

    for _, m := range o.migrations {
        if _, err := sqldb.Exec(m); err != nil {
            _ = sqldb.Close()
            return nil, fmt.Errorf("migration failed: %w", err)
        }
    }

    ctx, cancel := context.WithCancel(parent)
    db := &DB{
        sqldb:  sqldb,
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
    if err := db.execOnce(j.query, j.args...); err != nil {
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

func (db *DB) execOnce(query string, args ...any) error {
    ctx, cancel := context.WithTimeout(db.ctx, 10*time.Second)
    defer cancel()
    _, err := db.sqldb.ExecContext(ctx, query, args...)
    return err
}

func (db *DB) Enqueue(query string, args ...any) {
    db.jobs <- job{query: query, args: args}
}

func (db *DB) ExecSync(ctx context.Context, query string, args ...any) (sql.Result, error) {
    return db.sqldb.ExecContext(ctx, query, args...)
}

func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
    return db.sqldb.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
    return db.sqldb.QueryRowContext(ctx, query, args...)
}

func (db *DB) Close() error {
    db.cancel()
    close(db.jobs)
    db.wg.Wait()
    return db.sqldb.Close()
}

func BuildMySQLDSN(user, password, host string, port int, dbname string, charset string, parseTime bool, loc string) string {
    if charset == "" {
        charset = "utf8mb4"
    }
    parseTimeStr := "False"
    if parseTime {
        parseTimeStr = "True"
    }
    if loc == "" {
        loc = "Local"
    }
    return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
        user, password, host, port, dbname, charset, parseTimeStr, loc)
}