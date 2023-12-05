package pingpong

import (
	"fmt"
	"time"
)

type Ball struct {
	Hits int
}

func Player(name string, table chan *Ball) {
	for {
		ball := <-table
		ball.Hits++
		fmt.Printf("%s hits the ball for the %d-th time!\n", name, ball.Hits)
		time.Sleep(300 * time.Millisecond)
		table <- ball
	}
}

func MainExample() {
	table := make(chan *Ball)
	go Player("Joe", table)
	go Player("Ann", table)

	table <- new(Ball) // if commented out causes deadlock fatal error (all goroutines are asleep)
	time.Sleep(time.Second)
	<-table
}
