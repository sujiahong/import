package main

import (
	"fmt"
	"go/geo"
)

func main() {
	fmt.Println("=== 测试直线交点 CrossPoint ===")
	
	// 测试用例1: 两条相交的直线
	// 直线1: (0,0) -> (10,10)  y = x
	// 直线2: (0,10) -> (10,0)  y = 10 - x
	// 交点应该是 (5, 5)
	line1p1 := &geo.Point{X: 0, Y: 0}
	line1p2 := &geo.Point{X: 10, Y: 10}
	line2p1 := &geo.Point{X: 0, Y: 10}
	line2p2 := &geo.Point{X: 10, Y: 0}
	
	intersection := geo.CrossPoint(line1p1, line1p2, line2p1, line2p2)
	fmt.Printf("测试1 - 两条相交直线:\n")
	fmt.Printf("  直线1: (%v, %v) -> (%v, %v)\n", line1p1.X, line1p1.Y, line1p2.X, line1p2.Y)
	fmt.Printf("  直线2: (%v, %v) -> (%v, %v)\n", line2p1.X, line2p1.Y, line2p2.X, line2p2.Y)
	fmt.Printf("  交点: %v\n", intersection)
	fmt.Printf("  期望交点: (5, 5)\n")
	if intersection != nil {
		fmt.Printf("  结果: X=%.2f, Y=%.2f\n", intersection.X, intersection.Y)
	}
	
	// 测试用例2: 平行线
	// 直线1: (0,0) -> (10,10)  y = x
	// 直线2: (0,1) -> (10,11)  y = x + 1
	parallelLine1p1 := &geo.Point{X: 0, Y: 0}
	parallelLine1p2 := &geo.Point{X: 10, Y: 10}
	parallelLine2p1 := &geo.Point{X: 0, Y: 1}
	parallelLine2p2 := &geo.Point{X: 10, Y: 11}
	
	parallelIntersection := geo.CrossPoint(parallelLine1p1, parallelLine1p2, parallelLine2p1, parallelLine2p2)
	fmt.Printf("\n测试2 - 平行线:\n")
	fmt.Printf("  直线1: (%v, %v) -> (%v, %v)\n", parallelLine1p1.X, parallelLine1p1.Y, parallelLine1p2.X, parallelLine1p2.Y)
	fmt.Printf("  直线2: (%v, %v) -> (%v, %v)\n", parallelLine2p1.X, parallelLine2p1.Y, parallelLine2p2.X, parallelLine2p2.Y)
	fmt.Printf("  交点: %v (应该是nil)\n", parallelIntersection)
	
	// 测试用例3: 垂直线
	// 直线1: (5,0) -> (5,10)  x = 5
	// 直线2: (0,5) -> (10,5)  y = 5
	// 交点应该是 (5, 5)
	verticalLine1p1 := &geo.Point{X: 5, Y: 0}
	verticalLine1p2 := &geo.Point{X: 5, Y: 10}
	verticalLine2p1 := &geo.Point{X: 0, Y: 5}
	verticalLine2p2 := &geo.Point{X: 10, Y: 5}
	
	verticalIntersection := geo.CrossPoint(verticalLine1p1, verticalLine1p2, verticalLine2p1, verticalLine2p2)
	fmt.Printf("\n测试3 - 垂直线:\n")
	fmt.Printf("  直线1: (%v, %v) -> (%v, %v)\n", verticalLine1p1.X, verticalLine1p1.Y, verticalLine1p2.X, verticalLine1p2.Y)
	fmt.Printf("  直线2: (%v, %v) -> (%v, %v)\n", verticalLine2p1.X, verticalLine2p1.Y, verticalLine2p2.X, verticalLine2p2.Y)
	fmt.Printf("  交点: %v\n", verticalIntersection)
	fmt.Printf("  期望交点: (5, 5)\n")
	if verticalIntersection != nil {
		fmt.Printf("  结果: X=%.2f, Y=%.2f\n", verticalIntersection.X, verticalIntersection.Y)
	}
}
