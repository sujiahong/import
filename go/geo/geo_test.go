package geo_test

import (
	"go/geo"
	"go/my_util"
	"fmt"
	"testing"
	"math/rand"
	"time"
	"strings"
)

func TestTriangleInnerPoint(t *testing.T){
	rand.Seed(time.Now().Unix())
	p1 := &geo.Point{200, 400}
	p2 := &geo.Point{100, 300}
	p3 := &geo.Point{300, 100}
	p := geo.TriangleInnerPoint(p1, p2, p3)
	fmt.Println(p)
	b := geo.IsInTriangle([]*geo.Point{p1, p2, p3}, p)
	fmt.Println(b)
}

func TestPolygonInnerRandPoint(t *testing.T) {
	rand.Seed(time.Now().Unix())
	points_str := "279893.11,348601.43,279740.55,350344.96,279287.58,352035.49,278547.92,353621.70,277544.05,355055.35,276306.50,356292.90,274872.85,357296.78,273286.63,358036.43,271596.11,358489.40,269852.58,358641.96,268109.05,358489.40,266418.53,358036.43,264832.32,357296.78,263398.66,356292.90,262161.11,355055.35,261157.24,353621.70,260417.58,352035.49,259964.61,350344.96,259812.05,348601.43,259964.61,346857.90,260417.58,345167.38,261157.24,343581.17,262161.11,342147.52,263398.66,340909.97,264832.32,339906.09,266418.53,339166.44,268109.05,338713.46,269852.58,338560.91,271596.11,338713.46,273286.63,339166.44,274872.85,339906.09,276306.50,340909.97,277544.05,342147.52,278547.92,343581.17,279287.58,345167.38,279740.55,346857.90"
	//points_str := "100,300,200,500"
	str_arr := strings.Split(points_str, ",")
	if len(str_arr) & 1 == 1 { /////奇数
		return
	}
	var points []*geo.Point
	for i := 0; i < len(str_arr); i += 2 {
		p := &geo.Point{
			X: my_util.ToF64(str_arr[i]),
			Y: my_util.ToF64(str_arr[i+1]),
		}
		points = append(points, p)
	}
	count := 0
	fmt.Println("111    ", points, len(points))
	for i := 0; i < 10000; i++ {
		tmpPoint := geo.PolygonInnerRandPoint(points)
		if !geo.IsInPolygon(points, tmpPoint) {
			count++
			fmt.Println("222    ", tmpPoint, count)
		}
	}
}

func TestIsInPolygon(t *testing.T) {
	points_str := "279893.11,348601.43,279740.55,350344.96,279287.58,352035.49,278547.92,353621.70,277544.05,355055.35,276306.50,356292.90,274872.85,357296.78,273286.63,358036.43,271596.11,358489.40,269852.58,358641.96,268109.05,358489.40,266418.53,358036.43,264832.32,357296.78,263398.66,356292.90,262161.11,355055.35,261157.24,353621.70,260417.58,352035.49,259964.61,350344.96,259812.05,348601.43,259964.61,346857.90,260417.58,345167.38,261157.24,343581.17,262161.11,342147.52,263398.66,340909.97,264832.32,339906.09,266418.53,339166.44,268109.05,338713.46,269852.58,338560.91,271596.11,338713.46,273286.63,339166.44,274872.85,339906.09,276306.50,340909.97,277544.05,342147.52,278547.92,343581.17,279287.58,345167.38,279740.55,346857.90"
	str_arr := strings.Split(points_str, ",")
	if len(str_arr) & 1 == 1 { /////奇数
		return
	}
	var points []*geo.Point
	for i := 0; i < len(str_arr); i += 2 {
		p := &geo.Point{
			X: my_util.ToF64(str_arr[i]),
			Y: my_util.ToF64(str_arr[i+1]),
		}
		points = append(points, p)
	}
	b := geo.IsInPolygon(points, &geo.Point{
		X: 270418.006779,
		Y: 358592.48,
	})
	fmt.Println("bbbbb   ", b)
	b1 := geo.IsInTriangle(points, &geo.Point{
		X: 270418.006779,
		Y: 358592.48479200003,
	})
	fmt.Println("b111   ", b1)
}