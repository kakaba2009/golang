package main

import "testing"

func TestFibonacci0(t *testing.T) {
	x := 0
	var f int
	f = Fib(x)

	if f == 0 {
		t.Log("Success")
	} else {
		t.Error("Failure, expected 0, but got ", f)
	}
}

func TestFibonacci1(t *testing.T) {
	x := 1
	var f int
	f = Fib(x)

	if f == 1 {
		t.Log("Success")
	} else {
		t.Error("Failure, expected 1, but got ", f)
	}
}

func TestFibonacci2(t *testing.T) {
	x := 2
	var f int
	f = Fib(x)

	if f == 1 {
		t.Log("Success")
	} else {
		t.Error("Failure, expected 1, but got ", f)
	}
}
