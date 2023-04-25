package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program10"
)

func main() {
	err := program10.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
