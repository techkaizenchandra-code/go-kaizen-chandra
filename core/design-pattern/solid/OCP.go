package main

import (
	"fmt"
	"math"
)

// Shape interface - Open for extension, closed for modification
// Any new shape can implement this interface without changing existing code
type Shape interface {
	Area() float64
	Name() string
}

// Rectangle implementation
type Rectangle struct {
	Width  float64
	Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Name() string {
	return "Rectangle"
}

// Circle implementation
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Name() string {
	return "Circle"
}

// Triangle implementation
type Triangle struct {
	Base   float64
	Height float64
}

func (t Triangle) Area() float64 {
	return 0.5 * t.Base * t.Height
}

func (t Triangle) Name() string {
	return "Triangle"
}

// AreaCalculator - Closed for modification, open for extension
// This struct doesn't need to change when new shapes are added
type AreaCalculator struct{}

func (ac AreaCalculator) CalculateTotalArea(shapes []Shape) float64 {
	total := 0.0
	for _, shape := range shapes {
		total += shape.Area()
	}
	return total
}

func (ac AreaCalculator) PrintShapeAreas(shapes []Shape) {
	for _, shape := range shapes {
		fmt.Printf("%s Area: %.2f\n", shape.Name(), shape.Area())
	}
}

// TestOpenClosed demonstrates the Open/Closed Principle
func TestOpenClosed() {
	fmt.Println("=== Open/Closed Principle Demo ===")

	// Create various shapes
	shapes := []Shape{
		Rectangle{Width: 10, Height: 5},
		Circle{Radius: 7},
		Triangle{Base: 8, Height: 6},
		Rectangle{Width: 15, Height: 3},
		Circle{Radius: 4},
	}

	calculator := AreaCalculator{}

	// Print individual areas
	calculator.PrintShapeAreas(shapes)

	// Calculate total area
	totalArea := calculator.CalculateTotalArea(shapes)
	fmt.Printf("\nTotal Area of all shapes: %.2f\n", totalArea)

	fmt.Println("\n✓ Open/Closed Principle: Classes are open for extension but closed for modification")
	fmt.Println("✓ New shapes can be added without modifying AreaCalculator")
}
