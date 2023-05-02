package main

import (
	"log"

	"github.com/kakaba2009/golang/program1"
)

func main() {
	err := program1.Main()

	if err != nil {
		log.Println(err)
		return
	}
}
