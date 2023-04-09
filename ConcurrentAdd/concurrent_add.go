package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
)

func add(c1 <-chan int, c2 <-chan int, c3 chan<- int) {
	var msg, output int
	var ok1, ok2 bool
	for {
		msg, ok1 = <-c1
		if ok1 {
			output += msg
		}
		msg, ok2 = <-c2
		if ok2 {
			output += msg
		}
		if ok1 == false && ok2 == false {
			break
		}
		runtime.Gosched()
	}
	c3 <- output
	close(c3)
}

func input1(c1 chan<- int, arg int) {
	for i := 1; i <= arg; i++ {
		if i%2 == 0 {
			c1 <- i
		}
	}
	close(c1)
}

func input2(c2 chan<- int, arg int) {
	for i := 1; i <= arg; i++ {
		if i%2 == 1 {
			c2 <- i
		}
	}
	close(c2)
}

func main() {
	arg, _ := strconv.ParseInt(os.Args[1], 10, 32)

	c1 := make(chan int, 16)
	c2 := make(chan int, 16)
	c3 := make(chan int, 16)

	x := int(arg)

	go input1(c1, x)
	go input2(c2, x)
	go add(c1, c2, c3)

	sum := <-c3
	fmt.Println("Sum ", sum)
}
