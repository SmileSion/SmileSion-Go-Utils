package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "utils/db/xsqlite" // 替换成实际的 import 路径
)

func main() {
    // 配置数据库
    cfg := xsqlite.Config{
        DBPath:    "./demo.db",
        Workers:   2,
        QueueSize: 100,
    }

    // 初始化数据库，建表
    db, err := xsqlite.Open(context.Background(), cfg, xsqlite.WithMigrations([]string{
        `CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL,
            age INTEGER NOT NULL,
            created_at INTEGER NOT NULL
        );`,
    }))
    if err != nil {
        log.Fatalf("failed to open db: %v", err)
    }
    defer db.Close()

    // 异步写入几条记录
    for i := 0; i < 5; i++ {
        username := fmt.Sprintf("user%d", i)
        db.Enqueue(`INSERT INTO users(username, age, created_at) VALUES(?,?,?)`, username, 20+i, time.Now().Unix())
    }

    // 给异步 worker 一点时间写入
    time.Sleep(2 * time.Second)

    // 同步插入一条记录
    _, err = db.ExecSync(context.Background(), `INSERT INTO users(username, age, created_at) VALUES(?,?,?)`, "sync_user", 30, time.Now().Unix())
    if err != nil {
        log.Fatalf("failed to sync insert: %v", err)
    }

    // 同步查询所有用户
    rows, err := db.Query(context.Background(), `SELECT id, username, age, created_at FROM users ORDER BY id`)
    if err != nil {
        log.Fatalf("failed to query: %v", err)
    }
    defer rows.Close()

    fmt.Println("Users in DB:")
    for rows.Next() {
        var id int
        var username string
        var age int
        var createdAt int64
        if err := rows.Scan(&id, &username, &age, &createdAt); err != nil {
            log.Fatalf("scan failed: %v", err)
        }
        fmt.Printf("ID=%d, Username=%s, Age=%d, CreatedAt=%d\n", id, username, age, createdAt)
    }

    // 同步查询单条记录
    row := db.QueryRow(context.Background(), `SELECT username, age FROM users WHERE username=?`, "sync_user")
    var uname string
    var uage int
    if err := row.Scan(&uname, &uage); err != nil {
        log.Fatalf("queryrow failed: %v", err)
    }
    fmt.Printf("QueryRow -> Username=%s, Age=%d\n", uname, uage)
}
