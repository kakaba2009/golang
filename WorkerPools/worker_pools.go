package main

import (
	"fmt"
	"time"
)

func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		time.Sleep(time.Millisecond * 100)
		fmt.Println("worker", id, "finished job", j)
		results <- j * 2
	}
}

func main() {
	st := time.Now()
	const numJobs = 500
	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	for w := 1; w <= 10; w++ {
		go worker(w, jobs, results)
	}

	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= numJobs; a++ {
		<-results
	}
	et := time.Now()

	fmt.Println(et.Sub(st))
}
