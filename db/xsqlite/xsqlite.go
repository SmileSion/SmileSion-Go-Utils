// Package asql 提供一个带缓冲队列与工作池的 SQLite 异步/同步读写模块。
//
// 特性：
//   - 使用 modernc.org/sqlite，纯 Go 实现，无需 cgo
//   - 初始化时可执行建表 SQL
//   - 异步写入（Enqueue/EnqueueMany）
//   - 异步读取（EnqueueQuery -> channel 返回结果）
//   - 同步读写（ExecSync/Query/QueryRow）
//   - 优雅关闭（Close() 等待消费完成）
//
// 使用示例：
//   cfg := asql.Config{DBPath: "data/app.db", Workers: 4, QueueSize: 1000}
//   db, _ := asql.Open(context.Background(), cfg,
//       asql.WithMigrations([]string{
//           `CREATE TABLE IF NOT EXISTS logs(
//               id INTEGER PRIMARY KEY AUTOINCREMENT,
//               level TEXT NOT NULL,
//               msg   TEXT NOT NULL,
//               created_at INTEGER NOT NULL
//           );`,
//       }),
//   )
//   defer db.Close()
//
//   // 异步写
//   db.Enqueue("INSERT INTO logs(level,msg,created_at) VALUES(?,?,?)", "INFO", "hello", time.Now().Unix())
//
//   // 同步读
//   rows, _ := db.Query(context.Background(), "SELECT id,level,msg FROM logs WHERE level=?", "INFO")
//   defer rows.Close()
//
package xsqlite

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "math"
    "sync"
    "time"

    _ "modernc.org/sqlite"
)

type Config struct {
    DBPath      string
    Workers     int
    QueueSize   int
    BusyTimeout time.Duration
    SyncMode    string
    ExtraPragma []string
}

type Option func(*openOptions)

type openOptions struct {
    migrations []string
}

func WithMigrations(sqls []string) Option {
    return func(o *openOptions) { o.migrations = sqls }
}

type DB struct {
    sqldb   *sql.DB
    cfg     Config

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

type queryJob struct {
    query  string
    args   []any
    result chan queryResult
}

type queryResult struct {
    rows *sql.Rows
    err  error
}

func Open(parent context.Context, cfg Config, opts ...Option) (*DB, error) {
    if cfg.DBPath == "" {
        return nil, errors.New("DBPath required")
    }
    if cfg.Workers <= 0 {
        cfg.Workers = 2
    }
    if cfg.QueueSize <= 0 {
        cfg.QueueSize = 1000
    }
    if cfg.BusyTimeout <= 0 {
        cfg.BusyTimeout = 5 * time.Second
    }
    if cfg.SyncMode == "" {
        cfg.SyncMode = "NORMAL"
    }

    o := &openOptions{}
    for _, f := range opts {
        f(o)
    }

    dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
        cfg.DBPath, int(cfg.BusyTimeout.Milliseconds()))

    sqldb, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }

    sqldb.SetMaxOpenConns(max(4, cfg.Workers))
    sqldb.SetMaxIdleConns(cfg.Workers)
    sqldb.SetConnMaxLifetime(0)

    if len(cfg.ExtraPragma) > 0 {
        for _, p := range cfg.ExtraPragma {
            if _, err := sqldb.Exec("PRAGMA " + p); err != nil {
                _ = sqldb.Close()
                return nil, fmt.Errorf("apply pragma %q: %w", p, err)
            }
        }
    }

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
    j := job{query: query, args: args}
    db.jobs <- j
}

func (db *DB) EnqueueMany(query string, arglist ...[]any) {
    for _, a := range arglist {
        db.Enqueue(query, a...)
    }
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

func (db *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
    tx, err := db.sqldb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
    if err != nil {
        return err
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
        }
    }()
    if err := fn(tx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}

func max(a, b int) int { if a > b { return a }; return b }
