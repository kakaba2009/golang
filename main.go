package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program8"
)

func main() {
	err := program8.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
