package main

import (
	"log"

	"github.com/kakaba2009/golang/program12/server"
)

func main() {
	err := server.Main()

	if err != nil {
		log.Println(err)
		return
	}
}
