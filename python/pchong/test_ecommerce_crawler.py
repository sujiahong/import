#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
电商价格监控爬虫测试脚本
"""

import sys
import os
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

def test_imports():
    """测试模块导入"""
    print("1. 测试电商爬虫模块导入...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        print("  ✓ 电商爬虫模块导入成功")
        return True
    except ImportError as e:
        print(f"  ✗ 电商爬虫模块导入失败: {e}")
        return False
    except Exception as e:
        print(f"  ✗ 导入过程中出错: {e}")
        return False

def test_dependencies():
    """测试电商爬虫依赖包"""
    print("\n2. 测试电商爬虫依赖包...")
    
    dependencies = [
        ("selenium", "浏览器自动化"),
        ("webdriver_manager", "ChromeDriver管理"),
        ("beautifulsoup4", "HTML解析"),
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

def test_crawler_initialization():
    """测试爬虫初始化"""
    print("\n3. 测试电商爬虫初始化...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        
        crawler = EcommerceCrawler()
        
        # 检查基本属性
        required_attrs = ['name', 'platforms', 'config', 'logger']
        for attr in required_attrs:
            if hasattr(crawler, attr):
                print(f"  ✓ 属性 '{attr}' 存在")
            else:
                print(f"  ✗ 属性 '{attr}' 不存在")
                return False
        
        # 检查平台配置
        if isinstance(crawler.platforms, dict) and len(crawler.platforms) >= 4:
            print(f"  ✓ 平台配置正常 ({len(crawler.platforms)} 个平台)")
        else:
            print(f"  ✗ 平台配置异常")
            return False
        
        # 检查爬取配置
        if isinstance(crawler.config, dict) and 'max_pages' in crawler.config:
            print(f"  ✓ 爬取配置正常")
        else:
            print(f"  ✗ 爬取配置异常")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 爬虫初始化测试失败: {e}")
        return False

def test_price_extraction():
    """测试价格提取功能"""
    print("\n4. 测试价格提取功能...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        
        crawler = EcommerceCrawler()
        
        test_cases = [
            ("¥1299.00", 1299.0),
            ("￥899", 899.0),
            ("$99.99", 99.99),
            ("1,299.00元", 1299.0),
            ("特价：599", 599.0),
            ("价格：1,299.00", 1299.0),
            ("", None),
            ("免费", None),
        ]
        
        all_passed = True
        for text, expected in test_cases:
            result = crawler._extract_price(text)
            if result == expected:
                print(f"  ✓ '{text}' -> {result}")
            else:
                print(f"  ✗ '{text}' -> {result} (期望: {expected})")
                all_passed = False
        
        return all_passed
        
    except Exception as e:
        print(f"  ✗ 价格提取测试失败: {e}")
        return False

def test_sales_extraction():
    """测试销量提取功能"""
    print("\n5. 测试销量提取功能...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        
        crawler = EcommerceCrawler()
        
        test_cases = [
            ("已售1.2万件", 12000),
            ("销量：5000+", 5000),
            ("月销3万", 30000),
            ("评价 2.5w", 25000),
            ("100条评价", 100),
            ("", None),
            ("暂无销量", None),
        ]
        
        all_passed = True
        for text, expected in test_cases:
            result = crawler._extract_sales(text)
            if result == expected:
                print(f"  ✓ '{text}' -> {result}")
            else:
                print(f"  ✗ '{text}' -> {result} (期望: {expected})")
                all_passed = False
        
        return all_passed
        
    except Exception as e:
        print(f"  ✗ 销量提取测试失败: {e}")
        return False

def test_url_building():
    """测试URL构建功能"""
    print("\n6. 测试URL构建功能...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        
        crawler = EcommerceCrawler()
        
        # 测试淘宝URL构建
        taobao_platform = crawler.platforms['taobao']
        url = crawler._build_search_url(taobao_platform, "手机", 1)
        
        if url and "taobao.com" in url and "q=手机" in url:
            print(f"  ✓ 淘宝URL构建成功: {url[:50]}...")
        else:
            print(f"  ✗ 淘宝URL构建失败")
            return False
        
        # 测试京东URL构建
        jd_platform = crawler.platforms['jd']
        url = crawler._build_search_url(jd_platform, "笔记本电脑", 2)
        
        if url and "jd.com" in url and "keyword=笔记本电脑" in url:
            print(f"  ✓ 京东URL构建成功: {url[:50]}...")
        else:
            print(f"  ✗ 京东URL构建失败")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ URL构建测试失败: {e}")
        return False

def test_product_id_generation():
    """测试商品ID生成功能"""
    print("\n7. 测试商品ID生成功能...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        
        crawler = EcommerceCrawler()
        
        test_products = [
            {
                "platform": "淘宝",
                "title": "iPhone 15 Pro Max",
                "price": 8999.00
            },
            {
                "platform": "京东",
                "title": "华为Mate 60 Pro",
                "price": 6999.00
            },
            {
                "platform": "拼多多",
                "title": "小米14 Ultra",
                "price": 6499.00
            }
        ]
        
        generated_ids = set()
        for product in test_products:
            product_id = crawler._generate_product_id(product)
            if product_id and len(product_id) == 10:
                print(f"  ✓ 商品ID生成成功: {product_id}")
                generated_ids.add(product_id)
            else:
                print(f"  ✗ 商品ID生成失败")
                return False
        
        # 检查ID是否唯一
        if len(generated_ids) == len(test_products):
            print(f"  ✓ 商品ID唯一性验证通过")
        else:
            print(f"  ✗ 商品ID重复")
            return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 商品ID生成测试失败: {e}")
        return False

def test_demo_function():
    """测试演示功能"""
    print("\n8. 测试演示功能...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import demo_ecommerce_crawler
        
        print(" 正在运行演示程序...")
        mock_products = demo_ecommerce_crawler()
        
        if isinstance(mock_products, list) and len(mock_products) > 0:
            print(f"  ✓ 演示程序运行成功")
            print(f"     生成 {len(mock_products)} 个模拟商品")
            
            # 检查商品数据结构
            sample_product = mock_products[0]
            required_fields = ['product_id', 'platform', 'title', 'price', 'crawl_time']
            for field in required_fields:
                if field in sample_product:
                    print(f"  ✓ 商品包含字段: {field}")
                else:
                    print(f"  ✗ 商品缺少字段: {field}")
                    return False
            
            return True
        else:
            print(f"  ✗ 演示程序未返回数据")
            return False
        
    except Exception as e:
        print(f"  ✗ 演示功能测试失败: {e}")
        return False

def test_integration_with_base():
    """测试与基础爬虫类的集成"""
    print("\n9. 测试与基础爬虫类的集成...")
    
    try:
        from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
        from crawlers.base.base_crawler import BaseCrawler
        
        crawler = EcommerceCrawler()
        
        # 检查继承关系
        if isinstance(crawler, BaseCrawler):
            print(f"  ✓ 正确继承自BaseCrawler")
        else:
            print(f"  ✗ 未正确继承自BaseCrawler")
            return False
        
        # 检查基础方法
        base_methods = ['make_request', 'save_data', 'print_stats', 'run']
        for method in base_methods:
            if hasattr(crawler, method) and callable(getattr(crawler, method)):
                print(f"  ✓ 基础方法可用: {method}")
            else:
                print(f"  ✗ 基础方法不可用: {method}")
                return False
        
        return True
        
    except Exception as e:
        print(f"  ✗ 集成测试失败: {e}")
        return False

def main():
    """主测试函数"""
    print("=" * 60)
    print("电商价格监控爬虫 - 完整功能测试")
    print("=" * 60)
    
    test_results = []
    
    # 运行所有测试
    test_results.append(("模块导入", test_imports()))
    test_results.append(("依赖包", test_dependencies()))
    test_results.append(("爬虫初始化", test_crawler_initialization()))
    test_results.append(("价格提取", test_price_extraction()))
    test_results.append(("销量提取", test_sales_extraction()))
    test_results.append(("URL构建", test_url_building()))
    test_results.append(("商品ID生成", test_product_id_generation()))
    test_results.append(("演示功能", test_demo_function()))
    test_results.append(("基础集成", test_integration_with_base()))
    
    # 统计结果
    passed_count = sum(1 for _, result in test_results if result)
    total_count = len(test_results)
    
    print("\n" + "=" * 60)
    print("测试总结:")
    print(f"  总测试项: {total_count}")
    print(f"  通过项: {passed_count}")
    print(f"  失败项: {total_count - passed_count}")
    
    if passed_count == total_count:
        print("\n  ✅ 所有测试通过!")
        print("\n电商爬虫开发完成，功能正常。")
        print("下一步：集成到主控制程序并更新文档。")
        
        # 显示安装提示
        print("\n安装提示:")
        print("  1. 确保已安装Chrome浏览器")
        print("  2. 安装Selenium相关依赖:")
        print("     pip install selenium webdriver-manager")
        print("  3. 运行时可能需要VPN访问某些电商网站")
        
    else:
        print("\n  ❌ 部分测试失败，请修复上述问题。")
        
        # 显示失败详情
        print("\n失败详情:")
        for test_name, result in test_results:
            status = "✅" if result else "❌"
            print(f"  {status} {test_name}")
    
    print("\n" + "=" * 60)
    
    # 返回测试结果
    return passed_count == total_count


if __name__ == "__main__":
    try:
        success = main()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n\n测试被用户中断")
        sys.exit(0)
    except Exception as e:
        print(f"\n测试过程中出错: {e}")
        sys.exit(1)