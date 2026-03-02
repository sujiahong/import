package main

import (
	"fmt"
	"go/geo"
	"go/my_util"
)

func main() {
	// 测试随机数生成
	fmt.Println("=== 测试随机数生成 ===")
	for i := 0; i < 5; i++ {
		random := my_util.SafeRandRange(1, 100)
		fmt.Printf("随机数 %d: %d\n", i+1, random)
	}

	// 测试点是否在三角形内
	fmt.Println("\n=== 测试点是否在三角形内 ===")
	triangle := []*geo.Point{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 5, Y: 10},
	}
	pointInside := &geo.Point{X: 5, Y: 5}
	pointOutside := &geo.Point{X: 15, Y: 5}

	fmt.Printf("点 %v 是否在三角形内: %v\n", pointInside, geo.IsInTriangle(triangle, pointInside))
	fmt.Printf("点 %v 是否在三角形内: %v\n", pointOutside, geo.IsInTriangle(triangle, pointOutside))

	// 测试直线交点
	fmt.Println("\n=== 测试直线交点 ===")
	line1p1 := &geo.Point{X: 0, Y: 0}
	line1p2 := &geo.Point{X: 10, Y: 10}
	line2p1 := &geo.Point{X: 0, Y: 10}
	line2p2 := &geo.Point{X: 10, Y: 0}

	intersection := geo.CrossPoint(line1p1, line1p2, line2p1, line2p2)
	fmt.Printf("两条直线的交点: %v\n", intersection)

	// 测试多边形内随机点
	fmt.Println("\n=== 测试多边形内随机点 ===")
	polygon := []*geo.Point{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 10, Y: 10},
		{X: 0, Y: 10},
	}

	for i := 0; i < 3; i++ {
		randomPoint := geo.PolygonInnerRandPoint(polygon)
		fmt.Printf("多边形内随机点 %d: %v\n", i+1, randomPoint)
	}

	// 测试点是否在多边形内
	fmt.Println("\n=== 测试点是否在多边形内 ===")
	pointInPoly := &geo.Point{X: 5, Y: 5}
	pointOutPoly := &geo.Point{X: 15, Y: 5}

	fmt.Printf("点 %v 是否在多边形内: %v\n", pointInPoly, geo.IsInPolygon(polygon, pointInPoly))
	fmt.Printf("点 %v 是否在多边形内: %v\n", pointOutPoly, geo.IsInPolygon(polygon, pointOutPoly))

	fmt.Println("\n所有测试完成")
}
