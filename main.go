package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program2"
)

func main() {
	err := program2.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
