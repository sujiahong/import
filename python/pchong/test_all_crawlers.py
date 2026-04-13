#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试脚本 - 测试所有爬虫功能
"""

import sys
import os
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

def test_imports():
    """测试所有模块导入"""
    print("1. 测试模块导入...")
    
    modules_to_test = [
        ("config.settings", "配置模块"),
        ("crawlers.base.base_crawler", "基础爬虫类"),
        ("crawlers.news.news_crawler", "新闻爬虫"),
        ("utils.excel_exporter", "Excel导出模块"),
    ]
    
    all_passed = True
    
    for module_path, module_name in modules_to_test:
        try:
            __import__(module_path.replace('/', '.'))
            print(f"  ✓ {module_name} 导入成功")
        except ImportError as e:
            print(f"  ✗ {module_name} 导入失败: {e}")
            all_passed = False
        except Exception as e:
            print(f"  ✗ {module_name} 导入出错: {e}")
            all_passed = False
    
    return all_passed

def test_dependencies():
    """测试依赖包"""
    print("\n2. 测试依赖包...")
    
    dependencies = [
        ("requests", "HTTP请求库"),
        ("pandas", "数据处理库"),
        ("openpyxl", "Excel操作库"),
        ("beautifulsoup4", "HTML解析库"),
        ("fake-useragent", "User-Agent生成"),
    ]
    
    all_passed = True
    
    for package, description in dependencies:
        try:
            __import__(package.replace('-', '_'))
            print(f"  ✓ {description} ({package}) 可用")
        except ImportError:
            print(f"  ✗ {description} ({package}) 未安装")
            all_passed = False
    
    return all_passed

def test_base_crawler():
    """测试基础爬虫类"""
    print("\n3. 测试基础爬虫类...")
    
    try:
        from crawlers.base.base_crawler import BaseCrawler, SimpleWebCrawler
        
        # 测试基础功能
        crawler = SimpleWebCrawler("test", "https://httpbin.org/get")
        print(f"  ✓ 基础爬虫类创建成功")
        
        # 测试请求功能
        response = crawler.make_request("https://httpbin.org/get")
        if response and response.status_code == 200:
            print(f"  ✓ HTTP请求测试成功")
        else:
            print(f"  ✗ HTTP请求测试失败")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 基础爬虫测试失败: {e}")
        return False

def test_news_crawler():
    """测试新闻爬虫"""
    print("\n4. 测试新闻爬虫...")
    
    try:
        from crawlers.news.news_crawler import NewsCrawler
        
        crawler = NewsCrawler()
        print(f"  ✓ 新闻爬虫创建成功")
        
        # 测试配置
        if hasattr(crawler, 'news_sources') and len(crawler.news_sources) > 0:
            print(f"  ✓ 新闻源配置正常")
        else:
            print(f"  ✗ 新闻源配置异常")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 新闻爬虫测试失败: {e}")
        return False

def test_excel_exporter():
    """测试Excel导出功能"""
    print("\n5. 测试Excel导出功能...")
    
    try:
        from utils.excel_exporter import ExcelExporter
        
        exporter = ExcelExporter()
        print(f"  ✓ Excel导出器创建成功")
        
        # 测试数据
        test_data = [
            {"id": 1, "name": "测试1", "value": 100},
            {"id": 2, "name": "测试2", "value": 200},
        ]
        
        # 测试导出
        import tempfile
        with tempfile.TemporaryDirectory() as temp_dir:
            from pathlib import Path
            temp_path = Path(temp_dir)
            
            test_exporter = ExcelExporter(temp_path)
            output_file = test_exporter.export_to_excel(
                test_data, 
                "test_export.xlsx",
                sheet_name="测试数据"
            )
            
            if output_file.exists():
                print(f"  ✓ Excel文件生成成功")
                # 清理测试文件
                output_file.unlink(missing_ok=True)
            else:
                print(f"  ✗ Excel文件生成失败")
                return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ Excel导出测试失败: {e}")
        return False

def test_main_program():
    """测试主程序"""
    print("\n6. 测试主程序...")
    
    try:
        # 测试主程序导入
        import main
        
        print(f"  ✓ 主程序导入成功")
        
        # 测试爬虫管理器创建
        from main import CrawlerManager
        manager = CrawlerManager()
        
        # 测试列出爬虫
        crawlers = manager.list_crawlers()
        if isinstance(crawlers, list) and len(crawlers) > 0:
            print(f"  ✓ 爬虫列表获取成功 ({len(crawlers)} 个爬虫)")
        else:
            print(f"  ✗ 爬虫列表获取失败")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 主程序测试失败: {e}")
        return False

def run_demo():
    """运行演示程序"""
    print("\n7. 运行演示程序...")
    
    try:
        # 导入主程序
        import main
        
        # 运行演示
        result = main.demo_news_crawler()
        
        if result and result.get("success"):
            print(f"  ✓ 演示程序运行成功")
            print(f"     爬取数据: {result.get('data_count', 0)} 条")
            print(f"     Excel文件: {result.get('excel_file', 'N/A')}")
            return True
        else:
            print(f"  ✗ 演示程序运行失败")
            if result:
                print(f"     错误: {result.get('error', '未知错误')}")
            return False
            
    except Exception as e:
        print(f"  ✗ 演示程序运行出错: {e}")
        return False

def main():
    """主测试函数"""
    print("=" * 60)
    print("爬虫项目 - 完整功能测试")
    print("=" * 60)
    
    test_results = []
    
    # 运行所有测试
    test_results.append(("模块导入", test_imports()))
    test_results.append(("依赖包", test_dependencies()))
    test_results.append(("基础爬虫", test_base_crawler()))
    test_results.append(("新闻爬虫", test_news_crawler()))
    test_results.append(("Excel导出", test_excel_exporter()))
    test_results.append(("主程序", test_main_program()))
    
    # 统计结果
    passed_count = sum(1 for _, result in test_results if result)
    total_count = len(test_results)
    
    print("\n" + "=" * 60)
    print("测试总结:")
    print(f"  总测试项: {total_count}")
    print(f"  通过项: {passed_count}")
    print(f"  失败项: {total_count - passed_count}")
    
    if passed_count == total_count:
        print("\n  ✓ 所有基础测试通过!")
        
        # 询问是否运行演示程序
        print("\n是否运行演示程序? (y/n)")
        choice = input().strip().lower()
        
        if choice in ['y', 'yes', '是']:
            demo_result = run_demo()
            if demo_result:
                print("\n  ✓ 演示程序成功完成!")
                print("\n项目准备就绪，可以开始使用。")
                print("运行 'python main.py --list' 查看可用爬虫。")
                print("运行 'python main.py --demo' 运行演示程序。")
            else:
                print("\n  ✗ 演示程序运行失败，请检查配置。")
        else:
            print("\n项目基础测试完成，可以开始使用。")
        
    else:
        print("\n  ✗ 部分测试失败，请修复上述问题。")
        
        # 显示失败详情
        print("\n失败详情:")
        for test_name, result in test_results:
            status = "✓" if result else "✗"
            print(f"  {status} {test_name}")
    
    print("\n" + "=" * 60)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n测试被用户中断")
        sys.exit(0)
    except Exception as e:
        print(f"\n测试过程中出错: {e}")
        sys.exit(1)