package main

import (
	"fmt"
)

func Fib(x int) int {
	if x == 0 {
		return 0
	}

	if x == 1 {
		return 1
	}

	return Fib(x-1) + Fib(x-2)
}

func main() {
	x := []int{
		5,
		6,
		7,
	}

	for _, value := range x {
		fmt.Println(Fib(value))
	}
}
