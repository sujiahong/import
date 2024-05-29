package geo

import (
	"go/my_util"
	// "fmt"
)

type Point struct {
	X     float64
	Y     float64
}
func (p *Point)GetX() float64 {return p.X}
func (p *Point)GetY() float64 {return p.Y}

func SubVector(p1, p2 *Point) *Point {
	return &Point{
		X: p1.X - p2.X,
		Y: p1.Y  - p2.Y,
	}
}
func CrossProduct(p1, p2 *Point) float64 {
	return p1.X * p2.Y - p2.X * p1.Y
}
/////是否在三角形内
func IsInTriangle(points []*Point, tp *Point) bool {
	if len(points) <= 0 {
		return false
	}
	var nCurCrossProduct float64 = 0.0
	var nLastValue       float64 = 0.0
	for i := 0; i < len(points);  i++ {
		vP1 := SubVector(tp, points[i])
		nNextIndex := (i + 1) % len(points)
		vP2 := SubVector(points[nNextIndex], points[i])
		nCurCrossProduct = CrossProduct(vP1, vP2)
		if i > 0 && nCurCrossProduct * nLastValue <= 0 {
			return false
		}
		nLastValue = nCurCrossProduct
	}
	return true
}
////根据直线求交点
func CrossPoint(p1, p2, p3, p4 *Point) *Point {
	d1 := (p1.X - p2.X)*(p3.Y - p4.Y)
	d2 := (p3.X - p4.X)*(p1.Y - p2.Y)
	xp := d2 - d1
	yp := d1 - d2
	if xp == 0 || yp == 0 {
		return nil
	}
	xm := (p3.X - p4.X)*(p2.X * p1.Y - p1.X * p2.Y) - (p1.X - p2.X)*(p4.X * p3.Y - p3.X * p4.Y)
	ym := (p3.Y - p4.Y)*(p2.Y * p1.X - p1.Y * p2.X) - (p1.Y - p2.Y)*(p4.Y * p3.X - p3.Y * p4.X)
	return &Point{
		X: xm/xp,
		Y: ym/yp,
	}
}
func LineSegmentOnPoint(p1, p2 *Point) *Point {
	r1 := my_util.RandRange(0, 9999)
	r2 := 10000 - r1
	f1 := float64(r1) / 10000.0
	f2 := float64(r2) / 10000.0
	return &Point{
		X: f1*p2.X + f2*p1.X,
		Y: f1*p2.Y + f2*p1.Y,
	}
}
///三角形行内随机点
func TriangleInnerPoint(p1, p2, p3 *Point) *Point {
	r1 := my_util.RandRange(1, 9998)
	r2 := my_util.RandRange(1, 9999 - r1)
	r3 := 10000 - r1 - r2
	f1 := float64(r1) / 10000.0
	f2 := float64(r2) / 10000.0
	f3 := float64(r3) / 10000.0
	//fmt.Println(r1, r2, r3, f1, f2, f3)
	return &Point{
		X: f1*p1.X + f2*p2.X + f3*p3.X,
		Y: f1*p1.Y + f2*p2.Y + f3*p3.Y,
	}
}
//////多边形内随机点
func PolygonInnerRandPoint(points []*Point) *Point {
	ln := len(points)
	if ln == 2 {
		return LineSegmentOnPoint(points[0], points[1])
	}else if ln == 1 {
		return points[0]
	}else if ln == 0{
		return nil
	}
	triangleNum := (ln -3)+1
	k := my_util.RandRange(1, int64(triangleNum))
	p1 := points[0]
	p2 := points[int(k)]
	p3 := points[int(k)+1]
	return TriangleInnerPoint(p1, p2, p3)
}
/////判断点是否在多边形内(射线法)
func IsInPolygon(points []*Point, tp *Point) bool {
	ln := len(points)
	if ln < 3 {
		return false
	}
	var c_n int = 0
	var min, max float64
	for i := 0; i < ln; i++ {
		p1 := points[i]
		p2 := points[(i+1)%ln]
		if p1.Y == p2.Y {
			continue
		}
		if p1.Y <= p2.Y {
			min, max = p1.Y, p2.Y
		}else {
			min, max = p2.Y, p1.Y
		}
		if tp.Y < min || tp.Y >= max { ////点在线段之外，无交点  
			continue
		}
		dy1 := tp.Y - p1.Y
		dy2 := p2.Y - p1.Y
		dx := p2.X - p1.X
		x := dy1*dx/dy2 + p1.X
		if x > tp.X {
			c_n++
		}else if (x == tp.X){
			return true
		}
	}
	if (c_n & 1) == 1 {
		return true
	}else {
		return false
	}
}