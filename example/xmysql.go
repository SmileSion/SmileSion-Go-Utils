package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "utils/db/xmysql"
)

func main() {
    ctx := context.Background()

	dsn := xmysql.BuildMySQLDSN("root", "password", "127.0.0.1", 3306, "testdb", "utf8mb4", true, "Local")

    // 配置数据库连接（请改成你的实际账号密码和数据库名）
    cfg := xmysql.Config{
        DSN:       dsn,
        Workers:   4,
        QueueSize: 1000,
        MaxOpen:   20,
        MaxIdle:   10,
        MaxLife:   time.Hour,
    }

    // 初始化，建表
    db, err := xmysql.Open(ctx, cfg,
        xmysql.WithMigrations([]string{
            `CREATE TABLE IF NOT EXISTS logs (
                id BIGINT AUTO_INCREMENT PRIMARY KEY,
                level VARCHAR(20) NOT NULL,
                msg   TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );`,
        }),
    )
    if err != nil {
        log.Fatal("open db error:", err)
    }
    defer db.Close()

    // -------------------------------
    // 异步写入
    // -------------------------------
    db.Enqueue("INSERT INTO logs(level,msg) VALUES(?,?)", "INFO", "hello async")

    // -------------------------------
    // 同步写入
    // -------------------------------
    if _, err := db.ExecSync(ctx, "INSERT INTO logs(level,msg) VALUES(?,?)", "WARN", "hello sync"); err != nil {
        log.Println("exec sync error:", err)
    }

    // -------------------------------
    // 同步查询
    // -------------------------------
    rows, err := db.Query(ctx, "SELECT id, level, msg, created_at FROM logs ORDER BY id DESC LIMIT 5")
    if err != nil {
        log.Fatal("query error:", err)
    }
    defer rows.Close()

    fmt.Println("最近 5 条日志:")
    for rows.Next() {
        var (
            id        int64
            level     string
            msg       string
            createdAt time.Time
        )
        if err := rows.Scan(&id, &level, &msg, &createdAt); err != nil {
            log.Println("scan error:", err)
            continue
        }
        fmt.Printf("[%d] %s - %s (%s)\n", id, level, msg, createdAt.Format(time.RFC3339))
    }

    if err := rows.Err(); err != nil {
        log.Println("rows error:", err)
    }

    // -------------------------------
    // 同步单行查询
    // -------------------------------
    var count int
    if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM logs").Scan(&count); err != nil {
        log.Println("query row error:", err)
    } else {
        fmt.Println("日志总数:", count)
    }

    // 稍微等一下，让异步写也被消费掉
    time.Sleep(1 * time.Second)
}
