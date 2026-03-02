package main

import (
	"fmt"
	"go/geo"
)

func main() {
	// 测试 IsInTriangle 函数
	fmt.Println("=== 测试 IsInTriangle 函数 ===")
	triangle := []*geo.Point{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 5, Y: 10},
	}
	pointInside := &geo.Point{X: 5, Y: 5}
	pointOutside := &geo.Point{X: 15, Y: 5}
	emptyTriangle := []*geo.Point{}
	singlePoint := []*geo.Point{{X: 0, Y: 0}}

	fmt.Printf("点 %v 是否在三角形内: %v\n", pointInside, geo.IsInTriangle(triangle, pointInside))
	fmt.Printf("点 %v 是否在三角形内: %v\n", pointOutside, geo.IsInTriangle(triangle, pointOutside))
	fmt.Printf("空三角形测试: %v\n", geo.IsInTriangle(emptyTriangle, pointInside))
	fmt.Printf("单点三角形测试: %v\n", geo.IsInTriangle(singlePoint, pointInside))

	// 测试 CrossPoint 函数
	fmt.Println("\n=== 测试 CrossPoint 函数 ===")
	line1p1 := &geo.Point{X: 0, Y: 0}
	line1p2 := &geo.Point{X: 10, Y: 10}
	line2p1 := &geo.Point{X: 0, Y: 10}
	line2p2 := &geo.Point{X: 10, Y: 0}
	parallelLine1p1 := &geo.Point{X: 0, Y: 0}
	parallelLine1p2 := &geo.Point{X: 10, Y: 10}
	parallelLine2p1 := &geo.Point{X: 1, Y: 0}
	parallelLine2p2 := &geo.Point{X: 11, Y: 10}

	intersection := geo.CrossPoint(line1p1, line1p2, line2p1, line2p2)
	fmt.Printf("两条直线的交点: %v\n", intersection)

	parallelIntersection := geo.CrossPoint(parallelLine1p1, parallelLine1p2, parallelLine2p1, parallelLine2p2)
	fmt.Printf("两条平行线的交点: %v\n", parallelIntersection)

	// 测试 PolygonInnerRandPoint 函数
	fmt.Println("\n=== 测试 PolygonInnerRandPoint 函数 ===")
	line := []*geo.Point{{X: 0, Y: 0}, {X: 10, Y: 10}}
	singlePointPoly := []*geo.Point{{X: 5, Y: 5}}
	emptyPoly := []*geo.Point{}
	polygon := []*geo.Point{
		{X: 0, Y: 0},
		{X: 10, Y: 0},
		{X: 10, Y: 10},
		{X: 0, Y: 10},
	}

	linePoint := geo.PolygonInnerRandPoint(line)
	fmt.Printf("线段上的随机点: %v\n", linePoint)

	singlePointResult := geo.PolygonInnerRandPoint(singlePointPoly)
	fmt.Printf("单点多边形的随机点: %v\n", singlePointResult)

	emptyPointResult := geo.PolygonInnerRandPoint(emptyPoly)
	fmt.Printf("空多边形的随机点: %v\n", emptyPointResult)

	// 测试 IsInPolygon 函数
	fmt.Println("\n=== 测试 IsInPolygon 函数 ===")
	pointInPoly := &geo.Point{X: 5, Y: 5}
	pointOutPoly := &geo.Point{X: 15, Y: 5}

	fmt.Printf("点 %v 是否在多边形内: %v\n", pointInPoly, geo.IsInPolygon(polygon, pointInPoly))
	fmt.Printf("点 %v 是否在多边形内: %v\n", pointOutPoly, geo.IsInPolygon(polygon, pointOutPoly))

	// 测试边界情况
	fmt.Println("\n=== 测试边界情况 ===")
	// 测试空多边形
	fmt.Printf("空多边形测试: %v\n", geo.IsInPolygon(emptyPoly, pointInPoly))
	// 测试线段
	fmt.Printf("线段测试: %v\n", geo.IsInPolygon(line, pointInPoly))
}
