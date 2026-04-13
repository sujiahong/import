#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
更新主程序以支持电商爬虫演示
"""

import sys
from pathlib import Path

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

def add_ecommerce_demo():
    """在主程序中添加电商爬虫演示函数"""
    
    main_file = project_root / "main.py"
    
    # 读取现有内容
    with open(main_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 找到demo_news_crawler函数的位置
    demo_news_start = content.find("def demo_news_crawler():")
    if demo_news_start == -1:
        print("错误: 找不到demo_news_crawler函数")
        return False
    
    # 找到demo_news_crawler函数的结束位置
    demo_news_end = content.find("def main():", demo_news_start)
    if demo_news_end == -1:
        print("错误: 找不到main函数")
        return False
    
    # 在demo_news_crawler函数后添加demo_ecommerce_crawler函数
    new_function = """
def demo_ecommerce_crawler():
    \"\"\"演示电商爬虫\"\"\"
    print("演示: 运行电商价格监控爬虫")
    print("-" * 50)
    print("注意: 由于电商网站反爬虫严格，此演示使用模拟数据")
    print("实际使用时请确保已安装Chrome浏览器和ChromeDriver")
    print("-" * 50)
    
    from crawlers.ecommerce.ecommerce_crawler import demo_ecommerce_crawler as real_demo
    result = real_demo()
    
    return result


"""
    
    # 插入新函数
    new_content = content[:demo_news_end] + new_function + content[demo_news_end:]
    
    # 更新主函数中的参数解析部分
    # 找到demo参数的位置
    demo_param_start = new_content.find('        "--demo",')
    if demo_param_start != -1:
        # 找到该行的结束位置
        demo_line_end = new_content.find('\n', demo_param_start)
        demo_line = new_content[demo_param_start:demo_line_end]
        
        # 更新帮助文本
        new_demo_line = '        "--demo", \n        action="store_true",\n        help="运行演示程序（新闻爬虫）"'
        new_content = new_content.replace(demo_line, new_demo_line)
    
    # 在参数解析后添加电商演示参数
    # 找到所有参数定义结束的位置
    args_section_end = new_content.find('    args = parser.parse_args()')
    if args_section_end != -1:
        # 在参数解析前添加新的参数
        new_param = """    parser.add_argument(
        "--demo-ecommerce", 
        action="store_true",
        help="运行电商爬虫演示程序"
    )
    
"""
        # 插入新参数
        new_content = new_content[:args_section_end] + new_param + new_content[args_section_end:]
    
    # 在主函数处理逻辑中添加电商演示
    # 找到args.demo的处理位置
    demo_check_start = new_content.find('    elif args.demo:')
    if demo_check_start != -1:
        # 找到elif块的结束位置
        demo_check_end = new_content.find('\n    else:', demo_check_start)
        if demo_check_end == -1:
            demo_check_end = new_content.find('\n\n    ', demo_check_start)
        
        # 在args.demo处理前添加args.demo_ecommerce处理
        new_demo_check = """    elif args.demo_ecommerce:
        result = demo_ecommerce_crawler()
        if result:
            print(f"\\n演示完成!")
            print(f"生成 {len(result)} 个模拟商品")
        else:
            print(f"演示失败")
    """
        new_content = new_content[:demo_check_start] + new_demo_check + new_content[demo_check_start:]
    
    # 更新帮助信息中的示例
    help_examples_start = new_content.find('        print("  python main.py --demo')
    if help_examples_start != -1:
        help_examples_end = new_content.find('\n        print("  python main.py --run-all"', help_examples_start)
        if help_examples_end != -1:
            # 更新示例
            new_examples = """        print("  python main.py --demo                    # 运行新闻爬虫演示")
        print("  python main.py --demo-ecommerce        # 运行电商爬虫演示")"""
            new_content = new_content[:help_examples_start] + new_examples + new_content[help_examples_end:]
    
    # 写回文件
    with open(main_file, 'w', encoding='utf-8') as f:
        f.write(new_content)
    
    print("主程序已成功更新，添加了电商爬虫演示功能")
    return True

def update_crawler_list_display():
    """更新爬虫列表显示"""
    
    main_file = project_root / "main.py"
    
    # 读取现有内容
    with open(main_file, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 找到print_crawler_list函数
    print_func_start = content.find('def print_crawler_list(manager: CrawlerManager):')
    if print_func_start == -1:
        print("错误: 找不到print_crawler_list函数")
        return False
    
    # 找到函数的结束位置
    print_func_end = content.find('\n\n\n', print_func_start)
    if print_func_end == -1:
        print_func_end = len(content)
    
    # 替换整个函数
    new_print_func = """def print_crawler_list(manager: CrawlerManager):
    \"\"\"打印爬虫列表\"\"\"
    crawlers = manager.list_crawlers()
    
    print("\n可用爬虫列表:")
    print("-" * 80)
    
    for crawler in crawlers:
        status_icon = "✓" if crawler["status"] == "可用" else "⏳"
        print(f"{status_icon} [{crawler['id']:12s}] {crawler['name']}")
        print(f"   描述: {crawler['description']}")
        
        # 显示额外信息
        if crawler['id'] == 'ecommerce':
            print("   ⚠️  注意: 需要Chrome浏览器和ChromeDriver")
            print("   ⚠️  注意: 可能需要VPN访问某些网站")
        
        print()
"""
    
    new_content = content[:print_func_start] + new_print_func + content[print_func_end:]
    
    # 写回文件
    with open(main_file, 'w', encoding='utf-8') as f:
        f.write(new_content)
    
    print("爬虫列表显示已更新")
    return True

def main():
    """主函数"""
    print("开始更新主程序以支持电商爬虫...")
    
    try:
        # 1. 添加电商爬虫演示函数
        if not add_ecommerce_demo():
            print("❌ 添加电商演示函数失败")
            return False
        
        # 2. 更新爬虫列表显示
        if not update_crawler_list_display():
            print("❌ 更新爬虫列表显示失败")
            return False
        
        print("\n✅ 主程序更新完成!")
        print("\n新增功能:")
        print("  1. 电商爬虫演示: python main.py --demo-ecommerce")
        print("  2. 更新了爬虫列表显示，包含电商爬虫的注意事项")
        print("  3. 电商爬虫已集成到主控制程序")
        
        print("\n运行测试:")
        print("  python test_ecommerce_crawler.py")
        print("  python main.py --list")
        print("  python main.py --demo-ecommerce")
        
        return True
        
    except Exception as e:
        print(f"❌ 更新过程中出错: {e}")
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)