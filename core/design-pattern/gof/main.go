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
	fmt.Println("\nPrototype:")
	_ = creational.TestPrototype()
	fmt.Println("\nAbstract Factory:")
	_ = creational.TestAbstractFactory()
}
