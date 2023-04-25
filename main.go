package main

import (
	"fmt"

	"github.com/kakaba2009/golang/program6"
)

func main() {
	err := program6.Main()

	if err != nil {
		fmt.Println(err)
		return
	}
}
