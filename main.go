package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program7"
)

func main() {
	err := program7.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
