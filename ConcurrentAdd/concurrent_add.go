package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func add(c1 <-chan int, c2 <-chan int, c3 chan<- int) {
	var msg, output int
	for {
		select {
		case msg = <-c1:
			output += msg
			c3 <- output
		case msg = <-c2:
			output += msg
			c3 <- output
		default:
			time.After(time.Millisecond)
		}
	}
}

func input1(c1 chan<- int, arg int) {
	for i := 1; i <= arg; i++ {
		if i%2 == 0 {
			c1 <- i
		}
	}
}

func input2(c2 chan<- int, arg int) {
	for i := 1; i <= arg; i++ {
		if i%2 == 1 {
			c2 <- i
		}
	}
}

func main() {
	arg, _ := strconv.ParseInt(os.Args[1], 10, 32)

	c1 := make(chan int, 16)
	c2 := make(chan int, 16)
	c3 := make(chan int, 16)

	go input1(c1, int(arg))
	go input2(c2, int(arg))
	go add(c1, c2, c3)

	for {
		sum := <-c3
		fmt.Println("Sum ", sum)
	}
}
