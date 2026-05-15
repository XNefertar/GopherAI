package main

import "GopherAI/common/aihelper"

// seed-bench-sessions 是一个仅供压测使用的小工具：
// 在不启动整个 HTTP 服务的前提下，向全局 AIHelperManager
// 注入一批内存会话，用于本地校验 SeedUserSessions 行为。
//
// 实际压测流程更推荐使用 main.go 的 BENCH_MODE=session-list 启动方式，
// 这里只是作为基础冒烟工具保留。
func main() {
	manager := aihelper.GetGlobalManager()
	manager.SeedUserSessions("bench-user", 100)
}
