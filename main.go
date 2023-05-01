package main

import (
	"log"

	"github.com/kakaba2009/golang/program7"
)

func main() {
	err := program7.Main()

	if err != nil {
		log.Println(err)
		return
	}
}
