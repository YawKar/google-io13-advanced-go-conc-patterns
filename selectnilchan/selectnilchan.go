package selectnilchan

import (
	"fmt"
	"math/rand"
)

func MainExample() {
	a, b := make(chan string), make(chan string)
	go func() { a <- "a" }()
	go func() { b <- "b" }()

	if rand.Intn(2) == 0 {
		a = nil
		fmt.Println("a is nil")
	} else {
		b = nil
		fmt.Println("b is nil")
	}
	select {
	case s := <-a:
		fmt.Printf("got %s\n", s)
	case s := <-b:
		fmt.Printf("got %s\n", s)
	}
}
