package main

import (
	"fmt"
	"gof/creational"
)

func main() {
	fmt.Println("Singleton:")
	_ = creational.TestSingleton()
	fmt.Println("\nFactory:")
	_ = creational.TestFactory()
	fmt.Println("\nBuilder:")
	creational.TestBuilder()
}
