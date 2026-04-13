#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
最终演示脚本
展示所有10个爬虫的完整功能
"""

import sys
import time
from datetime import datetime
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

def demo_all_crawlers():
    """演示所有爬虫功能"""
    print("=" * 70)
    print("10个爬虫项目 - 最终演示")
    print("=" * 70)
    print(f"演示时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print()
    
    # 导入主控制程序
    try:
        from main import CrawlerManager
        
        print("🎯 步骤1: 创建爬虫管理器")
        print("-" * 50)
        manager = CrawlerManager()
        print("✅ 爬虫管理器创建成功")
        print(f"   输出目录: {manager.output_dir}")
        print()
        
        print("🎯 步骤2: 查看所有爬虫")
        print("-" * 50)
        crawlers = manager.list_crawlers()
        
        # 统计信息
        total_crawlers = len(crawlers)
        available_crawlers = sum(1 for c in crawlers if c["status"] == "可用")
        pending_crawlers = sum(1 for c in crawlers if c["status"] == "待开发")
        
        print(f"📊 爬虫统计:")
        print(f"   总爬虫数: {total_crawlers}")
        print(f"   可用爬虫: {available_crawlers}")
        print(f"   待开发爬虫: {pending_crawlers}")
        print()
        
        print("📋 爬虫列表:")
        print("-" * 50)
        for crawler in crawlers:
            status_icon = "✅" if crawler["status"] == "可用" else "⏳"
            print(f"{status_icon} [{crawler['id']:12s}] {crawler['name']}")
            print(f"    描述: {crawler['description']}")
            if crawler['id'] == 'ecommerce':
                print("    ⚠️  注意: 需要Chrome浏览器和ChromeDriver")
            print()
        
        print("🎯 步骤3: 运行爬虫演示")
        print("-" * 50)
        
        # 选择几个爬虫进行演示（避免运行所有，节省时间）
        demo_crawlers = ["news", "ecommerce", "social_media"]
        
        demo_results = {}
        
        for crawler_id in demo_crawlers:
            print(f"\n🚀 演示爬虫: {crawler_id}")
            
            try:
                # 获取爬虫信息
                crawler_info = manager.get_crawler_info(crawler_id)
                if not crawler_info:
                    print(f"   ❌ 爬虫 '{crawler_id}' 不存在")
                    continue
                
                if not crawler_info["class_available"]:
                    print(f"   ⏳ 爬虫 '{crawler_id}' 尚未开发")
                    continue
                
                print(f"   📝 名称: {crawler_info['name']}")
                print(f"   📋 描述: {crawler_info['description']}")
                
                # 运行爬虫（演示模式，使用最小参数）
                print(f"   ⏳ 开始爬取...")
                start_time = time.time()
                
                # 根据爬虫类型设置不同参数
                if crawler_id == "news":
                    result = manager.run_crawler(crawler_id, sources=["sina"], max_articles=3)
                elif crawler_id == "ecommerce":
                    result = manager.run_crawler(crawler_id, platforms=["taobao"], max_pages=1, products_per_page=5)
                elif crawler_id == "social_media":
                    result = manager.run_crawler(crawler_id, platforms=["weibo"], max_items=5)
                else:
                    result = manager.run_crawler(crawler_id)
                
                end_time = time.time()
                duration = end_time - start_time
                
                if "error" in result:
                    print(f"   ❌ 爬取失败: {result['error']}")
                else:
                    # 统计数据量
                    if "data" in result:
                        data_count = len(result["data"])
                    elif isinstance(result, dict) and "all_jobs" in result:
                        data_count = len(result["all_jobs"])
                    elif isinstance(result, dict) and "open_data" in result:
                        data_count = len(result["open_data"])
                    else:
                        data_count = 0
                    
                    print(f"   ✅ 爬取成功!")
                    print(f"     数据量: {data_count} 条")
                    print(f"     耗时: {duration:.2f} 秒")
                    
                    demo_results[crawler_id] = {
                        "success": True,
                        "data_count": data_count,
                        "duration": duration
                    }
                    
                    # 导出数据
                    try:
                        export_file = manager.export_data(result, f"demo_{crawler_id}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.xlsx")
                        print(f"     导出文件: {Path(export_file).name}")
                    except Exception as e:
                        print(f"     导出失败: {e}")
                
            except Exception as e:
                print(f"   ❌ 运行爬虫时出错: {e}")
                import traceback
                traceback.print_exc()
        
        print("\n" + "=" * 70)
        print("🎉 演示完成!")
        print("=" * 70)
        
        # 演示总结
        if demo_results:
            print("\n📊 演示总结:")
            print("-" * 50)
            
            total_data = sum(r["data_count"] for r in demo_results.values())
            total_time = sum(r["duration"] for r in demo_results.values())
            successful_crawlers = len(demo_results)
            
            print(f"✅ 成功演示爬虫: {successful_crawlers} 个")
            print(f"📈 总数据量: {total_data} 条")
            print(f"⏱️  总耗时: {total_time:.2f} 秒")
            print(f"📊 平均速度: {total_data/total_time:.1f} 条/秒" if total_time > 0 else "📊 平均速度: N/A")
            
            print("\n📋 详细结果:")
            for crawler_id, result in demo_results.items():
                print(f"   {crawler_id:12s}: {result['data_count']:3d} 条数据, {result['duration']:5.2f} 秒")
        
        print("\n🎯 下一步建议:")
        print("1. 查看 output/ 目录下的Excel文件")
        print("2. 运行 `python main.py --list` 查看所有爬虫")
        print("3. 运行 `python main.py --run <crawler_id>` 运行特定爬虫")
        print("4. 查看 `项目完成报告.md` 了解详细信息")
        
        print("\n" + "=" * 70)
        
    except ImportError as e:
        print(f"❌ 导入失败: {e}")
        print("请确保在项目根目录运行此脚本")
    except Exception as e:
        print(f"❌ 演示过程中出错: {e}")
        import traceback
        traceback.print_exc()

def quick_test():
    """快速测试所有爬虫导入"""
    print("\n🔧 快速导入测试")
    print("-" * 50)
    
    crawler_classes = [
        ("新闻爬虫", "crawlers.news.news_crawler", "NewsCrawler"),
        ("电商爬虫", "crawlers.ecommerce.ecommerce_crawler", "EcommerceCrawler"),
        ("社交媒体爬虫", "crawlers.social_media.social_crawler", "SocialMediaCrawler"),
        ("招聘爬虫", "crawlers.job.job_crawler", "JobCrawler"),
        ("金融爬虫", "crawlers.finance.finance_crawler", "FinanceCrawler"),
        ("房地产爬虫", "crawlers.real_estate.real_estate_crawler", "RealEstateCrawler"),
        ("学术爬虫", "crawlers.academic.academic_crawler", "AcademicCrawler"),
        ("旅游爬虫", "crawlers.travel.travel_crawler", "TravelCrawler"),
        ("视频爬虫", "crawlers.video.video_crawler", "VideoCrawler"),
        ("政府数据爬虫", "crawlers.government.government_crawler", "GovernmentCrawler"),
        ("基础爬虫类", "crawlers.base.base_crawler", "BaseCrawler"),
        ("Excel导出工具", "utils.excel_exporter", "ExcelExporter"),
        ("配置模块", "config.settings", "settings"),
    ]
    
    success_count = 0
    total_count = len(crawler_classes)
    
    for name, module_path, class_name in crawler_classes:
        try:
            # 动态导入
            exec(f"from {module_path} import {class_name}")
            print(f"✅ {name:20s} - 导入成功")
            success_count += 1
        except ImportError as e:
            print(f"❌ {name:20s} - 导入失败: {e}")
        except Exception as e:
            print(f"❌ {name:20s} - 错误: {e}")
    
    print(f"\n📊 导入测试结果: {success_count}/{total_count} 成功")
    
    if success_count == total_count:
        print("🎉 所有模块导入成功!")
    else:
        print("⚠️  部分模块导入失败，请检查依赖安装")

if __name__ == "__main__":
    print("10个爬虫项目 - 最终演示脚本")
    print("=" * 70)
    
    # 运行快速测试
    quick_test()
    
    # 询问是否运行完整演示
    print("\n" + "=" * 70)
    response = input("是否运行完整演示? (y/n): ").strip().lower()
    
    if response == 'y' or response == 'yes' or response == '是':
        print("\n开始完整演示...")
        demo_all_crawlers()
    else:
        print("\n跳过完整演示。")
        print("\n您仍然可以:")
        print("1. 运行 `python main.py --list` 查看所有爬虫")
        print("2. 运行 `python main.py --demo` 运行演示模式")
        print("3. 查看 `项目完成报告.md` 了解详细信息")
    
    print("\n" + "=" * 70)
    print("演示脚本结束。感谢使用！")
    print("=" * 70)