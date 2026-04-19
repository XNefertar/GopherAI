package main

import (
	"GopherAI/utils/myjwt"
	"fmt"
)

func main() {
	token, err := myjwt.GenerateToken(1, "bench-user")
	if err != nil {
		panic(err)
	}
	fmt.Println(token)
}
