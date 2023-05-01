package main

import (
	"log"

	"github.com/kakaba2009/golang/program12"
)

func main() {
	err := program12.Main()

	if err != nil {
		log.Println(err)
		return
	}
}
