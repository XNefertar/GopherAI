package main

import "GopherAI/common/aihelper"

func main() {
	manager := aihelper.GetGlobalManager()
	manager.SeedUserSessions("bench-user", 100)
}
