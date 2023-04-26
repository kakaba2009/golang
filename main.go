package main

import (
	"log"

	"github.com/kakaba2009/golang/program11"
)

func main() {
	err := program11.Main()

	if err != nil {
		log.Println(err)
		return
	}
}
