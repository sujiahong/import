#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试脚本 - 验证爬虫需求总结工具是否能正常运行
"""

import subprocess
import sys
import os

def test_dependencies():
    """测试依赖包是否安装"""
    print("1. 测试Python依赖包...")
    
    required_packages = ['pandas', 'openpyxl']
    
    for package in required_packages:
        try:
            __import__(package)
            print(f"  ✓ {package} 已安装")
        except ImportError:
            print(f"  ✗ {package} 未安装")
            print(f"    请运行: pip install {package}")
            return False
    
    return True

def test_script():
    """测试主脚本是否能正常运行"""
    print("\n2. 测试主脚本导入...")
    
    try:
        import create_crawler_requirements
        print("  ✓ 主脚本可以正常导入")
        
        # 测试需求数据创建
        requirements = create_crawler_requirements.create_crawler_requirements()
        print(f"  ✓ 成功创建 {len(requirements)} 个爬虫需求")
        
        # 检查数据完整性
        expected_fields = [
            "需求编号", "需求名称", "应用场景", "目标网站/平台",
            "主要数据类型", "技术复杂度", "反爬虫难度", "数据更新频率",
            "预计数据量", "数据用途", "关键技术要点", "建议工具/框架",
            "法律合规风险", "备注"
        ]
        
        for req in requirements:
            for field in expected_fields:
                if field not in req:
                    print(f"  ✗ 需求 '{req.get('需求名称', '未知')}' 缺少字段: {field}")
                    return False
        
        print("  ✓ 所有需求数据字段完整")
        return True
        
    except Exception as e:
        print(f"  ✗ 脚本导入失败: {e}")
        return False

def run_main():
    """运行主程序"""
    print("\n3. 运行主程序...")
    
    try:
        # 执行主程序
        result = subprocess.run(
            [sys.executable, "create_crawler_requirements.py"],
            capture_output=True,
            text=True,
            timeout=30
        )
        
        if result.returncode == 0:
            print("  ✓ 主程序运行成功")
            print(f"  输出:\n{result.stdout}")
            
            # 检查Excel文件是否生成
            if os.path.exists("爬虫需求总结.xlsx"):
                file_size = os.path.getsize("爬虫需求总结.xlsx")
                print(f"  ✓ Excel文件已生成，大小: {file_size/1024:.1f} KB")
                return True
            else:
                print("  ✗ Excel文件未生成")
                return False
        else:
            print(f"  ✗ 主程序运行失败 (返回码: {result.returncode})")
            print(f"  错误输出:\n{result.stderr}")
            return False
            
    except subprocess.TimeoutExpired:
        print("  ✗ 程序运行超时")
        return False
    except Exception as e:
        print(f"  ✗ 运行过程中出错: {e}")
        return False

def main():
    """主测试函数"""
    print("=" * 60)
    print("爬虫需求总结工具 - 测试脚本")
    print("=" * 60)
    
    # 记录测试结果
    tests_passed = 0
    tests_total = 3
    
    # 测试1：依赖包
    if test_dependencies():
        tests_passed += 1
    
    # 测试2：脚本导入
    if test_script():
        tests_passed += 1
    
    # 测试3：运行主程序
    if run_main():
        tests_passed += 1
    
    # 测试总结
    print("\n" + "=" * 60)
    print("测试总结:")
    print(f"  通过: {tests_passed}/{tests_total}")
    
    if tests_passed == tests_total:
        print("  ✓ 所有测试通过！工具可以正常运行。")
        print("\n下一步：")
        print("  1. 打开 '爬虫需求总结.xlsx' 查看生成的爬虫需求")
        print("  2. 如需修改需求，编辑 create_crawler_requirements.py 文件")
        print("  3. 重新运行 python create_crawler_requirements.py 更新Excel")
    else:
        print("  ✗ 部分测试失败，请检查上述错误信息。")
        return 1
    
    return 0

if __name__ == "__main__":
    sys.exit(main())