package main

import (
	"GopherAI/utils/myjwt"
	"fmt"
)

// gen-bench-token 是一个仅供压测使用的小工具：
// 直接为固定的 bench-user 生成一个长期有效的 JWT，
// 配合 chat-sessions.sh 等脚本作为鉴权头使用。
func main() {
	token, err := myjwt.GenerateToken(1, "bench-user")
	if err != nil {
		panic(err)
	}
	fmt.Println(token)
}
