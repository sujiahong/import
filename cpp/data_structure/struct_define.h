#ifndef _STRUCT_DEFINE_H_
#define _STRUCT_DEFINE_H_

#include <vector>

struct Point
{
    long p_x;
    long p_y;
};

struct Rectangle
{
    Point rect_point;
    unsigned long rect_width;
    unsigned long rect_leghth;
};

struct Circular
{
    Point center_point;
    unsigned long radius;
};

struct Triangle
{
    Point point_1;
    Point Point_2;
    Point Point_3;
};

struct Polygon////顺序排列的
{
    std::vector<Point> point_vec;
};


#endif