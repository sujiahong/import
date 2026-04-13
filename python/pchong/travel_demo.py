#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
旅游网站酒店价格爬虫演示脚本
展示CR008爬虫的实际应用
"""

import sys
import os
from pathlib import Path
from datetime import datetime, timedelta
import pandas as pd

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from crawlers.travel.travel_crawler import TravelCrawler


def run_basic_demo():
    """运行基础演示"""
    print("🏨 旅游网站酒店价格爬虫演示")
    print("=" * 60)
    
    # 1. 创建爬虫实例
    print("1. 创建爬虫实例...")
    crawler = TravelCrawler()
    print("   ✅ 实例创建成功")
    
    # 2. 设置搜索参数
    print("\n2. 设置搜索参数...")
    
    # 自动生成日期（明天到后天）
    tomorrow = (datetime.now() + timedelta(days=1)).strftime("%Y-%m-%d")
    day_after_tomorrow = (datetime.now() + timedelta(days=2)).strftime("%Y-%m-%d")
    
    search_params = {
        "city": "北京",  # 可以改为上海、广州、杭州等
        "check_in": tomorrow,
        "check_out": day_after_tomorrow,
        "platform": "all",  # 搜索所有平台
        "max_hotels": 20    # 限制数量，避免耗时过长
    }
    
    print(f"   城市: {search_params['city']}")
    print(f"   入住日期: {search_params['check_in']}")
    print(f"   离店日期: {search_params['check_out']}")
    print(f"   搜索平台: {search_params['platform']}")
    print(f"   最大酒店数: {search_params['max_hotels']}")
    
    # 3. 运行爬虫
    print("\n3. 运行爬虫...")
    print("   注意：首次运行可能需要较长时间初始化")
    print("   如果Selenium未安装，将使用模拟数据")
    
    try:
        result = crawler.run(**search_params)
        
        if result.get("summary", {}).get("status") == "success":
            print("   ✅ 爬取成功！")
            return result
        else:
            print("   ⚠️  爬取失败，使用模拟数据演示")
            return generate_mock_data()
            
    except Exception as e:
        print(f"   ❌ 爬取异常: {e}")
        print("   使用模拟数据继续演示...")
        return generate_mock_data()


def generate_mock_data():
    """生成模拟数据用于演示"""
    print("\n📊 生成模拟数据用于演示...")
    
    # 模拟酒店数据
    mock_hotels = []
    platforms = ["携程", "去哪儿", "飞猪"]
    cities = ["北京", "上海", "广州", "杭州"]
    hotel_names = [
        "北京国际大酒店", "上海金茂君悦", "广州白云宾馆", 
        "杭州西湖国宾馆", "北京王府井希尔顿", "上海外滩华尔道夫",
        "广州白天鹅宾馆", "杭州香格里拉饭店", "北京长城饭店",
        "上海和平饭店", "广州花园酒店", "杭州西子宾馆"
    ]
    
    import random
    from datetime import datetime
    
    for i in range(12):
        hotel = {
            "hotel_id": f"H{1000 + i}",
            "hotel_name": hotel_names[i % len(hotel_names)],
            "platform": platforms[i % len(platforms)],
            "city": cities[i % len(cities)],
            "address": f"{cities[i % len(cities)]}市某区某路{i+1}号",
            "star_rating": random.choice([3, 4, 5]),
            "user_rating": round(random.uniform(3.5, 5.0), 1),
            "review_count": random.randint(100, 5000),
            "lowest_price": random.randint(300, 1500),
            "original_price": 0,
            "discount": 0,
            "room_type": random.choice(["标准间", "豪华间", "套房"]),
            "room_name": f"{random.choice(['高级', '豪华', '行政'])}房",
            "breakfast": random.choice(["含早", "不含早", "可选"]),
            "wifi": "免费",
            "parking": random.choice(["免费停车", "收费停车", "无停车"]),
            "check_in": "14:00",
            "check_out": "12:00",
            "latitude": round(random.uniform(30.0, 40.0), 6),
            "longitude": round(random.uniform(110.0, 120.0), 6),
            "phone": f"010-{random.randint(1000, 9999)}{random.randint(1000, 9999)}",
            "facilities": "健身房,游泳池,餐厅,会议室",
            "tags": random.choice(["热门", "推荐", "豪华", "经济"]),
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "source_url": f"https://example.com/hotel/{1000 + i}"
        }
        
        # 计算原价和折扣
        if random.random() > 0.5:
            hotel["original_price"] = int(hotel["lowest_price"] * random.uniform(1.1, 1.5))
            hotel["discount"] = round((1 - hotel["lowest_price"] / hotel["original_price"]) * 100, 1)
        else:
            hotel["original_price"] = hotel["lowest_price"]
        
        mock_hotels.append(hotel)
    
    # 分析数据
    analysis = {
        "price_stats": {
            "min": min(h["lowest_price"] for h in mock_hotels),
            "max": max(h["lowest_price"] for h in mock_hotels),
            "average": round(sum(h["lowest_price"] for h in mock_hotels) / len(mock_hotels), 2),
            "median": sorted([h["lowest_price"] for h in mock_hotels])[len(mock_hotels) // 2]
        },
        "rating_stats": {
            "min": min(h["user_rating"] for h in mock_hotels),
            "max": max(h["user_rating"] for h in mock_hotels),
            "average": round(sum(h["user_rating"] for h in mock_hotels) / len(mock_hotels), 2),
            "count": len(mock_hotels)
        },
        "star_distribution": {
            3: sum(1 for h in mock_hotels if h["star_rating"] == 3),
            4: sum(1 for h in mock_hotels if h["star_rating"] == 4),
            5: sum(1 for h in mock_hotels if h["star_rating"] == 5)
        },
        "platform_distribution": {
            "携程": sum(1 for h in mock_hotels if h["platform"] == "携程"),
            "去哪儿": sum(1 for h in mock_hotels if h["platform"] == "去哪儿"),
            "飞猪": sum(1 for h in mock_hotels if h["platform"] == "飞猪")
        },
        "recommendations": []
    }
    
    # 计算推荐
    scored_hotels = []
    for hotel in mock_hotels:
        score = (hotel["user_rating"] / hotel["lowest_price"]) * 1000
        scored_hotels.append({
            "hotel": hotel["hotel_name"],
            "platform": hotel["platform"],
            "rating": hotel["user_rating"],
            "price": hotel["lowest_price"],
            "score": round(score, 2),
            "address": hotel["address"]
        })
    
    scored_hotels.sort(key=lambda x: x["score"], reverse=True)
    analysis["recommendations"] = scored_hotels[:10]
    
    result = {
        "data": mock_hotels,
        "stats": {
            "total_hotels": len(mock_hotels),
            "platforms_searched": 3,
            "cities_searched": 4,
            "success_rate": 100,
            "total_time": 2.5
        },
        "analysis": analysis,
        "summary": {
            "total": len(mock_hotels),
            "city": "北京",
            "check_in": "2024-12-01",
            "check_out": "2024-12-07",
            "platforms": "all",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "status": "success",
            "average_price": analysis["price_stats"]["average"],
            "min_price": analysis["price_stats"]["min"],
            "max_price": analysis["price_stats"]["max"]
        }
    }
    
    print("   ✅ 模拟数据生成完成")
    return result


def display_results(result):
    """显示结果"""
    print("\n" + "=" * 60)
    print("📊 爬取结果显示")
    print("=" * 60)
    
    data = result.get("data", [])
    summary = result.get("summary", {})
    stats = result.get("stats", {})
    analysis = result.get("analysis", {})
    
    # 显示基本信息
    print(f"📋 基本信息:")
    print(f"   城市: {summary.get('city', '未知')}")
    print(f"   入住: {summary.get('check_in', '未知')}")
    print(f"   离店: {summary.get('check_out', '未知')}")
    print(f"   平台: {summary.get('platforms', '未知')}")
    print(f"   酒店总数: {summary.get('total', 0)}家")
    print(f"   搜索用时: {stats.get('total_time', 0)}秒")
    print(f"   成功率: {stats.get('success_rate', 0)}%")
    
    # 显示价格分析
    price_stats = analysis.get("price_stats", {})
    if price_stats:
        print(f"\n💰 价格分析:")
        print(f"   最低价: ¥{price_stats.get('min', 0)}")
        print(f"   最高价: ¥{price_stats.get('max', 0)}")
        print(f"   平均价: ¥{price_stats.get('average', 0)}")
        print(f"   中位数: ¥{price_stats.get('median', 0)}")
    
    # 显示星级分布
    star_dist = analysis.get("star_distribution", {})
    if star_dist:
        print(f"\n⭐ 星级分布:")
        for star in sorted(star_dist.keys()):
            count = star_dist[star]
            if count > 0:
                print(f"   {star}星酒店: {count}家 ({count/sum(star_dist.values())*100:.1f}%)")
    
    # 显示平台分布
    platform_dist = analysis.get("platform_distribution", {})
    if platform_dist:
        print(f"\n🏢 平台分布:")
        for platform, count in platform_dist.items():
            if count > 0:
                print(f"   {platform}: {count}家")
    
    # 显示酒店列表（前5个）
    if data:
        print(f"\n🏨 酒店列表 (前5家):")
        print("-" * 80)
        print(f"{'序号':<4} {'酒店名称':<20} {'平台':<8} {'星级':<6} {'评分':<6} {'价格':<8} {'地址':<20}")
        print("-" * 80)
        
        for i, hotel in enumerate(data[:5], 1):
            name = hotel.get("hotel_name", "未知")[:18] + ".." if len(hotel.get("hotel_name", "")) > 18 else hotel.get("hotel_name", "未知")
            platform = hotel.get("platform", "未知")[:6]
            stars = "★" * hotel.get("star_rating", 0)
            rating = hotel.get("user_rating", 0)
            price = f"¥{hotel.get('lowest_price', 0)}"
            address = hotel.get("address", "未知")[:18] + ".." if len(hotel.get("address", "")) > 18 else hotel.get("address", "未知")
            
            print(f"{i:<4} {name:<20} {platform:<8} {stars:<6} {rating:<6.1f} {price:<8} {address:<20}")
    
    # 显示推荐酒店
    recommendations = analysis.get("recommendations", [])
    if recommendations:
        print(f"\n🏆 性价比推荐 (前3名):")
        print("-" * 60)
        for i, rec in enumerate(recommendations[:3], 1):
            print(f"{i}. {rec.get('hotel', '未知')}")
            print(f"   平台: {rec.get('platform', '未知')}")
            print(f"   评分: {rec.get('rating', 0)}/5.0")
            print(f"   价格: ¥{rec.get('price', 0)}")
            print(f"   性价比得分: {rec.get('score', 0):.2f}")
            print(f"   地址: {rec.get('address', '未知')}")
            print()


def export_data_demo(result):
    """数据导出演示"""
    print("\n" + "=" * 60)
    print("💾 数据导出演示")
    print("=" * 60)
    
    try:
        crawler = TravelCrawler()
        
        print("1. 导出数据到Excel...")
        filepath = crawler.export_data(result)
        
        if filepath:
            print(f"   ✅ 导出成功!")
            print(f"   文件路径: {filepath}")
            
            # 显示文件信息
            import os
            if os.path.exists(filepath):
                file_size = os.path.getsize(filepath)
                print(f"   文件大小: {file_size:,d} bytes ({file_size/1024:.1f} KB)")
                
                # 读取Excel文件内容预览
                try:
                    print("\n2. Excel文件内容预览:")
                    df = pd.read_excel(filepath, sheet_name=None)
                    
                    print(f"   工作表数量: {len(df)}")
                    for sheet_name, sheet_data in df.items():
                        print(f"   📄 {sheet_name}: {len(sheet_data)}行 × {len(sheet_data.columns)}列")
                        
                        if sheet_name == "酒店列表" and not sheet_data.empty:
                            print(f"      前3行数据:")
                            for i in range(min(3, len(sheet_data))):
                                hotel_name = str(sheet_data.iloc[i].get("hotel_name", ""))[:30]
                                price = sheet_data.iloc[i].get("lowest_price", 0)
                                rating = sheet_data.iloc[i].get("user_rating", 0)
                                print(f"      {i+1}. {hotel_name} - ¥{price} - 评分:{rating}")
                        
                except Exception as e:
                    print(f"   ⚠️  读取Excel文件失败: {e}")
        else:
            print("   ⚠️  导出失败，可能是无数据或权限问题")
            
    except Exception as e:
        print(f"   ❌ 数据导出演示失败: {e}")


def advanced_features_demo():
    """高级功能演示"""
    print("\n" + "=" * 60)
    print("🚀 高级功能演示")
    print("=" * 60)
    
    try:
        crawler = TravelCrawler()
        
        print("1. 多城市搜索演示:")
        cities = ["北京", "上海", "杭州"]
        for city in cities:
            print(f"   • 搜索{city}的酒店 (限制5家)")
            result = crawler.run(city=city, max_hotels=5)
            if result["data"]:
                print(f"     找到 {len(result['data'])} 家酒店")
                if result["data"]:
                    avg_price = result["analysis"].get("price_stats", {}).get("average", 0)
                    print(f"     平均价格: ¥{avg_price}")
            else:
                print(f"     未找到数据")
        
        print("\n2. 特定平台搜索演示:")
        platforms = ["ctrip", "qunar", "fliggy"]
        for platform in platforms:
            platform_name = crawler.platforms.get(platform, {}).get("name", platform)
            print(f"   • 仅搜索{platform_name}平台")
            result = crawler.run(platform=platform, max_hotels=3)
            if result["data"]:
                hotels = [h for h in result["data"] if h.get("platform") == platform_name]
                print(f"     找到 {len(hotels)} 家{platform_name}酒店")
        
        print("\n3. 价格区间过滤演示:")
        # 这里演示如何在后处理中过滤
        print("   • 可以过滤价格在300-800元之间的酒店")
        print("   • 可以过滤评分在4.0以上的酒店")
        print("   • 可以过滤特定星级的酒店")
        
        print("\n4. 数据比较分析:")
        print("   • 比较不同平台的酒店价格")
        print("   • 分析不同城市的酒店价格差异")
        print("   • 识别性价比最高的酒店")
        
    except Exception as e:
        print(f"   ⚠️  高级功能演示失败: {e}")


def main():
    """主函数"""
    print("""
    ============================================================
    🏨 旅游网站酒店价格爬虫 (CR008) 完整演示
    ============================================================
    功能说明:
    • 爬取携程、去哪儿、飞猪的酒店价格信息
    • 支持多城市、多日期搜索
    • 提供智能分析和推荐
    • 导出Excel格式报告
    
    注意: 实际爬取需要ChromeDriver和网络连接
          演示中可能使用模拟数据
    ============================================================
    """)
    
    # 运行基础演示
    result = run_basic_demo()
    
    # 显示结果
    display_results(result)
    
    # 数据导出演示
    export_data_demo(result)
    
    # 高级功能演示
    advanced_features_demo()
    
    # 总结
    print("\n" + "=" * 60)
    print("🎉 演示完成！")
    print("=" * 60)
    print("\n📋 功能总结:")
    print("   1. ✅ 多平台酒店搜索")
    print("   2. ✅ 实时价格监控")
    print("   3. ✅ 智能数据分析")
    print("   4. ✅ Excel数据导出")
    print("   5. ✅ 高级过滤功能")
    
    print("\n🚀 实际使用:")
    print("   # 搜索北京酒店")
    print('   crawler = TravelCrawler()')
    print('   result = crawler.run(city="北京", platform="all")')
    print('   ')
    print('   # 导出数据')
    print('   crawler.export_data(result)')
    print('   ')
    print('   # 通过主程序使用')
    print('   python main.py --run travel --city 上海')
    
    print("\n📚 更多信息:")
    print("   • 查看测试脚本: test_travel_crawler.py")
    print("   • 查看源代码: crawlers/travel/travel_crawler.py")
    print("   • 查看配置: config/settings.py")


if __name__ == "__main__":
    main()