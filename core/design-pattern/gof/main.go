package main

import (
	"fmt"
	creational2 "gof/src/creational"
)

func main() {

	fmt.Println("Singleton:")
	_ = creational2.TestSingleton()

	fmt.Println("\nFactory:")
	_ = creational2.TestFactory()

	fmt.Println("\nBuilder:")
	creational2.TestBuilder()

	fmt.Println("\nPrototype:")
	_ = creational2.TestPrototype()

	fmt.Println("\nAbstract Factory:")
	_ = creational2.TestAbstractFactory()

}
