package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program9"
)

func main() {
	err := program9.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
