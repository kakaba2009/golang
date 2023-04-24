package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program5"
)

func main() {
	err := program5.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
