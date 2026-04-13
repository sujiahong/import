#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
更新主程序中的爬虫注册状态
将所有爬虫设置为可用状态
"""

import re
from pathlib import Path

# 读取主程序文件
main_file = Path(__file__).parent / "main.py"
with open(main_file, 'r', encoding='utf-8') as f:
    content = f.read()

# 爬虫映射关系
crawler_mapping = {
    "social_media": "SocialMediaCrawler",
    "job": "JobCrawler",
    "real_estate": "RealEstateCrawler",
    "finance": "FinanceCrawler",
    "academic": "AcademicCrawler",
    "travel": "TravelCrawler",
    "video": "VideoCrawler",
    "government": "GovernmentCrawler"
}

# 更新所有爬虫的class字段
for crawler_id, class_name in crawler_mapping.items():
    # 查找爬虫定义
    pattern = rf'self\.crawlers\["{crawler_id}"\]\s*=\s*{{\s*"name":\s*"[^"]+",\s*"class":\s*(None|[^,]+),\s*"description":'
    
    # 替换None为对应的类名
    new_pattern = f'self.crawlers["{crawler_id}"] = {{\n        "name": "{crawler_id}爬虫名称",\n        "class": {class_name},\n        "description":'
    
    # 更精确的替换
    content = re.sub(
        rf'self\.crawlers\["{crawler_id}"\]\s*=\s*{{[^}}]+"class":\s*\KNone(?=[^}}]+}})',
        class_name,
        content,
        flags=re.DOTALL
    )

# 修复描述信息（保持原有的描述）
def fix_descriptions(content):
    """修复爬虫描述信息"""
    descriptions = {
        "social_media": "社交媒体情感分析爬虫",
        "job": "招聘网站职位信息爬虫",
        "real_estate": "房地产市场价格爬虫",
        "finance": "股票金融数据爬虫",
        "academic": "学术论文文献爬虫",
        "travel": "旅游网站酒店价格爬虫",
        "video": "视频平台热门内容爬虫",
        "government": "政府公开数据爬虫"
    }
    
    for crawler_id, description in descriptions.items():
        # 查找并修复描述
        pattern = rf'self\.crawlers\["{crawler_id}"\]\s*=\s*{{[^}}]+"name":\s*"[^"]+",[^}}]+"description":\s*"[^"]*"(?=[^}}]+}})'
        
        # 构建新的描述
        name = description
        new_desc = f'self.crawlers["{crawler_id}"] = {{\n        "name": "{name}",\n        "class": {crawler_mapping.get(crawler_id, "None")},\n        "description": "{description}",'
        
        # 使用更简单的方法：直接替换整个块
        start_pattern = f'self.crawlers["{crawler_id}"] = {{'
        if start_pattern in content:
            # 找到开始位置
            start_idx = content.find(start_pattern)
            # 找到结束位置（下一个爬虫开始或函数结束）
            end_patterns = [f'self.crawlers["', '    def list_crawlers', '    def get_crawler_info']
            end_idx = len(content)
            
            for pattern in end_patterns:
                idx = content.find(pattern, start_idx + len(start_pattern))
                if idx != -1 and idx < end_idx:
                    end_idx = idx
            
            # 提取并替换这个块
            old_block = content[start_idx:end_idx]
            
            # 构建新的块
            new_block = f'''        # {description}
        self.crawlers["{crawler_id}"] = {{
            "name": "{description}",
            "class": {crawler_mapping.get(crawler_id, "None")},
            "description": "{descriptions.get(crawler_id, '')}",
            "config": specific_config.{crawler_id.upper()}
        }}
        
'''
            
            content = content[:start_idx] + new_block + content[end_idx:]
    
    return content

# 应用修复
content = fix_descriptions(content)

# 写回文件
with open(main_file, 'w', encoding='utf-8') as f:
    f.write(content)

print("✅ 爬虫注册状态已更新完成!")
print("   已将以下爬虫设置为可用状态:")
for crawler_id, class_name in crawler_mapping.items():
    print(f"   - {crawler_id}: {class_name}")

# 验证更新
print("\n📋 验证更新:")
with open(main_file, 'r', encoding='utf-8') as f:
    lines = f.readlines()
    for i, line in enumerate(lines):
        if 'self.crawlers[' in line and 'class' in line:
            print(f"   {i+1}: {line.strip()}")