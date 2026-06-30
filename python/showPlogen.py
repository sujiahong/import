import matplotlib.pyplot as plt
from shapely.geometry import Point, Polygon

# 多边形顶点（按顺序）
polygon_vertices = [
    (11449623, 47433420),
    (11350184, 47440636),
    (11242293, 47403116),
    (11148463, 47324088),
    (11102469, 47254824),
    (11169560, 47211248),
    (11208530, 47271248),
    (11282150, 47333752),
    (11360090, 47361248),
    (11442360, 47353752)   # 闭合时需注意：多边形会自动闭合
]

# 待判断的点
point_a = Point(11255747, 47311335)
point_target = Point(11442360, 47353753)

# 创建多边形对象
polygon = Polygon(polygon_vertices)

# 判断点是否在内部
print(f"点 (11255747,47311335) 在多边形内: {polygon.contains(point_a)}")
print(f"target (11442360,47353753) 在多边形内: {polygon.contains(point_target)}")
print(f"target 是否在多边形边界上: {polygon.touches(point_target)}")

# 绘图
fig, ax = plt.subplots(figsize=(8, 6))
x, y = polygon.exterior.xy
ax.fill(x, y, alpha=0.25, fc='lightblue', ec='black', linewidth=2, label='多边形')
ax.plot(point_a.x, point_a.y, 'ro', markersize=8, label='点 (11255747,47311335)')
ax.plot(point_target.x, point_target.y, 'gs', markersize=8, label='target (11442360,47353753)')

# 标注坐标值
ax.annotate(f'({point_a.x}, {point_a.y})', (point_a.x, point_a.y),
            textcoords="offset points", xytext=(10,10), fontsize=8)
ax.annotate(f'({point_target.x}, {point_target.y})', (point_target.x, point_target.y),
            textcoords="offset points", xytext=(10,-15), fontsize=8)

ax.set_aspect('equal')
ax.set_xlabel('X')
ax.set_ylabel('Y')
ax.set_title('多边形与点')
ax.legend()
plt.tight_layout()
plt.show()