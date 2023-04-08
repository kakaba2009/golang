package main

import "fmt"

func main() {
	messages := make(chan string, 1024)

	messages <- "buffered"
	messages <- "channel"

	fmt.Println(<-messages)
	fmt.Println(<-messages)
}
