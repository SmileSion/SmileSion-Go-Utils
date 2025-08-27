package main

import (
	"fmt"
	"utils/uuid"
)

func main() {
	u1 := uuid.MustV1()
	fmt.Println("UUID v1:", u1)

	u4 := uuid.MustV4()
	fmt.Println("UUID v4:", u4)

	ns := uuid.MustV4() // 随机生成命名空间
	u5 := uuid.MustV5(ns, "my-name")
	fmt.Println("UUID v5:", u5)

	// 校验
	fmt.Println("v5 是否有效:", uuid.IsValidUUID(string(u5))) // true
	fmt.Println("非法 UUID 校验:", uuid.IsValidUUID("1234-abcd")) // false
}
