package main

import (
	"fmt"
)

func doSomething() error {
	return nil
}

func main() {
	_ = doSomething()
	fmt.Println("hello")
}
