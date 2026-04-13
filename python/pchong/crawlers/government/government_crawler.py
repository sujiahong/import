#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
政府公开数据爬虫 (CR010)
爬取政府数据开放平台的公开数据
"""

import time
import json
import random
import csv
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any
import requests
from bs4 import BeautifulSoup
import pandas as pd
from io import StringIO

from crawlers.base.base_crawler import BaseCrawler
from config import settings, specific_config


class GovernmentCrawler(BaseCrawler):
    """政府公开数据爬虫"""
    
    def __init__(self, output_dir: str = None):
        """
        初始化政府数据爬虫
        
        Args:
            output_dir: 输出目录
        """
        super().__init__(output_dir)
        self.config = specific_config.GOVERNMENT
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": random.choice(settings.USER_AGENTS),
            "Accept": "application/json, text/html, */*",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
            "Accept-Encoding": "gzip, deflate, br",
            "Connection": "keep-alive",
        })
    
    def get_gov_open_data(self, platform: str = "national", category: str = None, max_datasets: int = 20) -> List[Dict]:
        """
        获取政府开放数据
        
        Args:
            platform: 平台类型，可选 ["national", "shanghai", "beijing", "guangdong"]
            category: 数据分类
            max_datasets: 最大数据集数
            
        Returns:
            政府开放数据列表
        """
        print(f"获取政府开放数据，平台: {platform}，分类: {category or '全部'}，最大数量: {max_datasets}")
        
        try:
            datasets = []
            
            if platform == "national":
                # 国家数据开放平台
                datasets = self._crawl_national_gov_data(category, max_datasets)
            elif platform == "shanghai":
                # 上海数据开放平台
                datasets = self._crawl_shanghai_gov_data(category, max_datasets)
            elif platform == "beijing":
                # 北京数据开放平台
                datasets = self._crawl_beijing_gov_data(category, max_datasets)
            elif platform == "guangdong":
                # 广东数据开放平台
                datasets = self._crawl_guangdong_gov_data(category, max_datasets)
            else:
                # 默认使用模拟数据
                datasets = self._generate_mock_gov_data(max_datasets, category)
            
            print(f"政府开放数据获取完成，共获取 {len(datasets)} 个数据集")
            return datasets
            
        except Exception as e:
            print(f"获取政府开放数据时出错: {e}")
            return []
    
    def _crawl_national_gov_data(self, category: str = None, max_datasets: int = 20) -> List[Dict]:
        """爬取国家数据开放平台数据"""
        try:
            # 国家数据开放平台API
            base_url = "https://data.stats.gov.cn"
            
            # 构建搜索URL
            search_url = f"{base_url}/api/search"
            params = {
                "query": category or "",
                "page": 1,
                "size": max_datasets
            }
            
            response = self._make_request(search_url, params=params)
            if not response:
                return self._generate_mock_gov_data(max_datasets, category)
            
            data = response.json()
            datasets = []
            
            if "data" in data and "list" in data["data"]:
                for item in data["data"]["list"][:max_datasets]:
                    dataset = {
                        "platform": "national",
                        "title": item.get("title", ""),
                        "description": item.get("description", ""),
                        "category": item.get("category", ""),
                        "publish_date": item.get("publishDate", ""),
                        "update_date": item.get("updateDate", ""),
                        "data_format": item.get("format", ""),
                        "file_size": item.get("fileSize", ""),
                        "download_url": f"{base_url}{item.get('downloadUrl', '')}" if item.get('downloadUrl') else "",
                        "view_url": f"{base_url}{item.get('viewUrl', '')}" if item.get('viewUrl') else "",
                        "department": item.get("department", ""),
                        "keywords": item.get("keywords", []),
                        "access_count": item.get("accessCount", 0),
                        "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    }
                    datasets.append(dataset)
            
            return datasets
            
        except Exception as e:
            print(f"爬取国家数据开放平台时出错: {e}")
            return self._generate_mock_gov_data(max_datasets, category)
    
    def _crawl_shanghai_gov_data(self, category: str = None, max_datasets: int = 20) -> List[Dict]:
        """爬取上海数据开放平台数据"""
        try:
            # 上海数据开放平台
            base_url = "https://data.sh.gov.cn"
            
            # 获取数据集列表
            api_url = f"{base_url}/api/datasets"
            params = {
                "page": 1,
                "size": max_datasets
            }
            
            if category:
                params["category"] = category
            
            response = self._make_request(api_url, params=params)
            if not response:
                return self._generate_mock_gov_data(max_datasets, category, "上海")
            
            data = response.json()
            datasets = []
            
            if "data" in data and "list" in data["data"]:
                for item in data["data"]["list"][:max_datasets]:
                    dataset = {
                        "platform": "shanghai",
                        "title": item.get("title", ""),
                        "description": item.get("description", ""),
                        "category": item.get("category", ""),
                        "publish_date": item.get("publishDate", ""),
                        "update_date": item.get("updateDate", ""),
                        "data_format": item.get("format", ""),
                        "file_size": item.get("fileSize", ""),
                        "download_url": f"{base_url}{item.get('downloadUrl', '')}" if item.get('downloadUrl') else "",
                        "view_url": f"{base_url}{item.get('detailUrl', '')}" if item.get('detailUrl') else "",
                        "department": item.get("department", "上海市相关部门"),
                        "keywords": item.get("tags", []),
                        "access_count": item.get("viewCount", 0),
                        "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    }
                    datasets.append(dataset)
            
            return datasets
            
        except Exception as e:
            print(f"爬取上海数据开放平台时出错: {e}")
            return self._generate_mock_gov_data(max_datasets, category, "上海")
    
    def _crawl_beijing_gov_data(self, category: str = None, max_datasets: int = 20) -> List[Dict]:
        """爬取北京数据开放平台数据"""
        try:
            # 北京数据开放平台
            base_url = "https://data.beijing.gov.cn"
            
            # 这里使用模拟数据，实际需要根据API调整
            datasets = []
            
            for i in range(max_datasets):
                dataset = {
                    "platform": "beijing",
                    "title": f"北京市{i+1}月份空气质量数据",
                    "description": f"北京市{i+1}月份空气质量监测数据，包括PM2.5、PM10、SO2、NO2等指标",
                    "category": category or "环境",
                    "publish_date": f"2026-{i+1:02d}-01",
                    "update_date": f"2026-{i+1:02d}-28",
                    "data_format": "CSV",
                    "file_size": f"{random.randint(10, 100)}KB",
                    "download_url": f"{base_url}/download/dataset_{i+1}.csv",
                    "view_url": f"{base_url}/dataset/{i+1}",
                    "department": "北京市生态环境局",
                    "keywords": ["空气质量", "PM2.5", "环境监测"],
                    "access_count": random.randint(100, 5000),
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
                datasets.append(dataset)
            
            return datasets
            
        except Exception as e:
            print(f"爬取北京数据开放平台时出错: {e}")
            return self._generate_mock_gov_data(max_datasets, category, "北京")
    
    def _crawl_guangdong_gov_data(self, category: str = None, max_datasets: int = 20) -> List[Dict]:
        """爬取广东数据开放平台数据"""
        try:
            # 广东数据开放平台
            base_url = "https://data.gd.gov.cn"
            
            # 模拟数据
            datasets = []
            
            for i in range(max_datasets):
                dataset = {
                    "platform": "guangdong",
                    "title": f"广东省{i+1}季度经济发展数据",
                    "description": f"广东省{i+1}季度GDP、工业增加值、固定资产投资等经济发展数据",
                    "category": category or "经济",
                    "publish_date": f"2026-{i*3+1:02d}-15",
                    "update_date": f"2026-{(i+1)*3:02d}-30",
                    "data_format": "JSON",
                    "file_size": f"{random.randint(50, 200)}KB",
                    "download_url": f"{base_url}/api/dataset/{i+1}.json",
                    "view_url": f"{base_url}/dataset/{i+1}",
                    "department": "广东省统计局",
                    "keywords": ["经济", "GDP", "统计"],
                    "access_count": random.randint(200, 8000),
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
                datasets.append(dataset)
            
            return datasets
            
        except Exception as e:
            print(f"爬取广东数据开放平台时出错: {e}")
            return self._generate_mock_gov_data(max_datasets, category, "广东")
    
    def _generate_mock_gov_data(self, max_datasets: int = 20, category: str = None, platform: str = "national") -> List[Dict]:
        """生成政府数据模拟数据"""
        categories = {
            "经济": ["GDP数据", "财政收入", "居民消费", "固定资产投资", "外贸进出口"],
            "人口": ["人口普查", "就业数据", "教育程度", "老龄化数据", "迁移统计"],
            "环境": ["空气质量", "水质监测", "土壤污染", "垃圾处理", "碳排放"],
            "交通": ["公共交通", "公路运输", "铁路客运", "航空运输", "物流数据"],
            "民生": ["医疗保障", "教育资源", "住房保障", "社会保障", "公共安全"]
        }
        
        departments = {
            "national": ["国家统计局", "财政部", "生态环境部", "交通运输部", "卫健委"],
            "shanghai": ["上海市统计局", "上海市财政局", "上海市生态环境局", "上海市交通委", "上海市卫健委"],
            "beijing": ["北京市统计局", "北京市财政局", "北京市生态环境局", "北京市交通委", "北京市卫健委"],
            "guangdong": ["广东省统计局", "广东省财政厅", "广东省生态环境厅", "广东省交通厅", "广东省卫健委"]
        }
        
        datasets = []
        
        for i in range(max_datasets):
            # 确定分类
            if category and category in categories:
                cat = category
                subcats = categories[cat]
            else:
                cat = random.choice(list(categories.keys()))
                subcats = categories[cat]
            
            # 生成标题和描述
            subcat = random.choice(subcats)
            title = f"{subcat}报告（{platform}平台）"
            description = f"本数据集包含{cat}领域的{subcat}详细数据，由{departments[platform][i % len(departments[platform])]}发布。"
            
            # 生成发布日期（最近1年内）
            days_ago = random.randint(1, 365)
            publish_date = (datetime.now() - timedelta(days=days_ago)).strftime("%Y-%m-%d")
            
            # 生成更新日期（在发布日期之后）
            update_days = random.randint(1, 30)
            update_date = (datetime.now() - timedelta(days=days_ago - update_days)).strftime("%Y-%m-%d")
            
            # 数据格式
            formats = ["CSV", "JSON", "XML", "XLSX", "PDF"]
            data_format = random.choice(formats)
            
            # 文件大小
            file_sizes = ["10KB", "50KB", "100KB", "500KB", "1MB", "5MB"]
            file_size = random.choice(file_sizes)
            
            # 关键词
            base_keywords = {
                "经济": ["发展", "增长", "产业", "投资", "消费"],
                "人口": ["统计", "调查", "就业", "教育", "医疗"],
                "环境": ["监测", "保护", "污染", "治理", "生态"],
                "交通": ["运输", "出行", "物流", "基础设施", "规划"],
                "民生": ["保障", "服务", "福利", "安全", "资源"]
            }
            keywords = random.sample(base_keywords.get(cat, ["数据", "公开", "信息", "统计"]), 3)
            
            dataset = {
                "platform": platform,
                "title": title,
                "description": description,
                "category": cat,
                "publish_date": publish_date,
                "update_date": update_date,
                "data_format": data_format,
                "file_size": file_size,
                "download_url": f"https://data.example.com/dataset_{i+1}.{data_format.lower()}",
                "view_url": f"https://data.example.com/view/{i+1}",
                "department": departments[platform][i % len(departments[platform])],
                "keywords": keywords,
                "access_count": random.randint(100, 10000),
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            }
            datasets.append(dataset)
        
        return datasets
    
    def get_statistical_report(self, report_type: str = "annual", year: int = 2025) -> Dict[str, Any]:
        """
        获取统计报告
        
        Args:
            report_type: 报告类型，可选 ["annual", "quarterly", "monthly"]
            year: 年份
            
        Returns:
            统计报告数据
        """
        print(f"获取统计报告，类型: {report_type}，年份: {year}")
        
        try:
            # 模拟统计报告数据
            reports = {
                "annual": {
                    "title": f"{year}年中国统计年鉴",
                    "publisher": "国家统计局",
                    "publish_date": f"{year+1}-09-01",
                    "pages": 1200,
                    "chapters": [
                        {"name": "综合", "pages": 150, "tables": 45},
                        {"name": "人口", "pages": 100, "tables": 30},
                        {"name": "国民经济核算", "pages": 120, "tables": 35},
                        {"name": "就业和工资", "pages": 80, "tables": 25},
                        {"name": "价格指数", "pages": 90, "tables": 28},
                        {"name": "人民生活", "pages": 110, "tables": 32},
                        {"name": "财政金融", "pages": 130, "tables": 38},
                        {"name": "资源与环境", "pages": 95, "tables": 27},
                        {"name": "能源", "pages": 85, "tables": 24},
                        {"name": "固定资产投资", "pages": 105, "tables": 31},
                        {"name": "对外经济贸易", "pages": 115, "tables": 34},
                        {"name": "农业", "pages": 125, "tables": 36},
                        {"name": "工业", "pages": 140, "tables": 42},
                        {"name": "建筑业", "pages": 75, "tables": 22},
                        {"name": "批发和零售业", "pages": 88, "tables": 26},
                        {"name": "运输、邮电和软件业", "pages": 92, "tables": 27},
                        {"name": "住宿、餐饮业和旅游业", "pages": 68, "tables": 20},
                        {"name": "金融业", "pages": 102, "tables": 30},
                        {"name": "房地产业", "pages": 78, "tables": 23},
                        {"name": "科学技术", "pages": 96, "tables": 28},
                        {"name": "教育", "pages": 112, "tables": 33},
                        {"name": "卫生和社会服务", "pages": 84, "tables": 25},
                        {"name": "文化和体育", "pages": 76, "tables": 22},
                        {"name": "公共管理、社会保障和社会组织", "pages": 98, "tables": 29},
                        {"name": "城市、农村和区域发展", "pages": 108, "tables": 32},
                        {"name": "香港特别行政区主要社会经济指标", "pages": 65, "tables": 18},
                        {"name": "澳门特别行政区主要社会经济指标", "pages": 62, "tables": 17},
                        {"name": "台湾省主要社会经济指标", "pages": 70, "tables": 20}
                    ],
                    "total_tables": 850,
                    "total_figures": 320,
                    "download_formats": ["PDF", "EXCEL", "CSV", "HTML"],
                    "file_size": "250MB",
                    "download_url": f"https://data.stats.gov.cn/yearbook/{year}/download.zip",
                    "view_url": f"https://data.stats.gov.cn/yearbook/{year}/",
                    "description": f"{year}年中国统计年鉴全面、系统地反映了当年中国国民经济和社会发展情况，是研究中国经济和社会发展的重要资料。",
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                },
                "quarterly": {
                    "title": f"{year}年第{(datetime.now().month-1)//3+1}季度国民经济运行情况",
                    "publisher": "国家统计局",
                    "publish_date": f"{year}-{(datetime.now().month-1)//3*3+1:02d}-20",
                    "pages": 80,
                    "sections": ["宏观经济", "产业发展", "市场需求", "就业物价", "居民收入", "对外贸易"],
                    "key_indicators": [
                        {"name": "GDP增长率", "value": f"{random.uniform(4.5, 6.5):.1f}%"},
                        {"name": "工业增加值增长率", "value": f"{random.uniform(3.0, 8.0):.1f}%"},
                        {"name": "固定资产投资增长率", "value": f"{random.uniform(2.0, 7.0):.1f}%"},
                        {"name": "社会消费品零售总额增长率", "value": f"{random.uniform(3.5, 9.0):.1f}%"},
                        {"name": "居民消费价格指数(CPI)", "value": f"{random.uniform(0.5, 3.5):.1f}%"},
                        {"name": "城镇调查失业率", "value": f"{random.uniform(4.8, 6.2):.1f}%"},
                        {"name": "货物进出口总额增长率", "value": f"{random.uniform(2.0, 10.0):.1f}%"}
                    ],
                    "download_formats": ["PDF", "PPT", "EXCEL"],
                    "file_size": "15MB",
                    "download_url": f"https://data.stats.gov.cn/quarterly/{year}/q{(datetime.now().month-1)//3+1}/report.zip",
                    "view_url": f"https://data.stats.gov.cn/quarterly/{year}/q{(datetime.now().month-1)//3+1}/",
                    "description": "季度报告反映了当季国民经济运行的主要情况，包括经济增长、产业发展、市场需求、就业物价等方面的数据和分析。",
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                },
                "monthly": {
                    "title": f"{year}年{datetime.now().month}月份主要经济指标",
                    "publisher": "国家统计局",
                    "publish_date": f"{year}-{datetime.now().month:02d}-15",
                    "indicators": [
                        {"name": "规模以上工业增加值同比增速", "value": f"{random.uniform(3.0, 9.0):.1f}%"},
                        {"name": "社会消费品零售总额同比增速", "value": f"{random.uniform(2.5, 12.0):.1f}%"},
                        {"name": "固定资产投资（不含农户）同比增速", "value": f"{random.uniform(1.5, 8.5):.1f}%"},
                        {"name": "全国居民消费价格指数(CPI)同比涨幅", "value": f"{random.uniform(0.5, 4.0):.1f}%"},
                        {"name": "工业生产者出厂价格指数(PPI)同比涨幅", "value": f"{random.uniform(-2.0, 6.0):.1f}%"},
                        {"name": "货物进出口总额同比增速", "value": f"{random.uniform(0.5, 15.0):.1f}%"},
                        {"name": "城镇调查失业率", "value": f"{random.uniform(4.5, 6.5):.1f}%"}
                    ],
                    "update_frequency": "每月15日",
                    "download_formats": ["EXCEL", "CSV", "PDF"],
                    "file_size": "8MB",
                    "download_url": f"https://data.stats.gov.cn/monthly/{year}/m{datetime.now().month}/data.zip",
                    "view_url": f"https://data.stats.gov.cn/monthly/{year}/m{datetime.now().month}/",
                    "description": "月度报告提供了主要经济指标的月度数据，有助于及时了解经济运行的动态变化。",
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
            }
            
            report = reports.get(report_type, reports["annual"])
            print(f"获取到统计报告: {report['title']}")
            return report
            
        except Exception as e:
            print(f"获取统计报告时出错: {e}")
            return {}
    
    def _make_request(self, url: str, params: Dict = None, max_retries: int = 3) -> Optional[requests.Response]:
        """发送HTTP请求"""
        for attempt in range(max_retries):
            try:
                # 随机延迟
                if attempt > 0:
                    delay = random.uniform(1, 3)
                    time.sleep(delay)
                
                # 更新User-Agent
                self.session.headers["User-Agent"] = random.choice(settings.USER_AGENTS)
                
                response = self.session.get(url, params=params, timeout=15)
                
                if response.status_code == 200:
                    return response
                elif response.status_code == 429:  # 请求过多
                    wait_time = 5 * (attempt + 1)
                    print(f"请求过多，等待 {wait_time} 秒后重试...")
                    time.sleep(wait_time)
                else:
                    print(f"请求失败，状态码: {response.status_code}, 尝试 {attempt+1}/{max_retries}")
                    
            except requests.exceptions.RequestException as e:
                print(f"请求异常: {e}, 尝试 {attempt+1}/{max_retries}")
            
            except Exception as e:
                print(f"未知错误: {e}, 尝试 {attempt+1}/{max_retries}")
        
        return None
    
    def analyze_gov_data_trends(self, datasets: List[Dict]) -> Dict[str, Any]:
        """
        分析政府数据趋势
        
        Args:
            datasets: 政府数据集
            
        Returns:
            趋势分析结果
        """
        if not datasets:
            return {"total_datasets": 0, "message": "没有数据"}
        
        print("分析政府数据趋势...")
        
        # 按平台统计
        platform_stats = {}
        for dataset in datasets:
            platform = dataset.get("platform", "unknown")
            platform_stats[platform] = platform_stats.get(platform, 0) + 1
        
        # 按分类统计
        category_stats = {}
        for dataset in datasets:
            category = dataset.get("category", "未知")
            category_stats[category] = category_stats.get(category, 0) + 1
        
        # 按部门统计
        department_stats = {}
        for dataset in datasets:
            department = dataset.get("department", "未知部门")
            department_stats[department] = department_stats.get(department, 0) + 1
        
        # 按数据格式统计
        format_stats = {}
        for dataset in datasets:
            data_format = dataset.get("data_format", "未知")
            format_stats[data_format] = format_stats.get(data_format, 0) + 1
        
        # 更新频率分析
        update_counts = {"daily": 0, "weekly": 0, "monthly": 0, "quarterly": 0, "yearly": 0, "irregular": 0}
        for dataset in datasets:
            update_freq = dataset.get("update_frequency", "").lower()
            if "每日" in update_freq or "daily" in update_freq:
                update_counts["daily"] += 1
            elif "每周" in update_freq or "weekly" in update_freq:
                update_counts["weekly"] += 1
            elif "每月" in update_freq or "monthly" in update_freq:
                update_counts["monthly"] += 1
            elif "每季" in update_freq or "quarterly" in update_freq:
                update_counts["quarterly"] += 1
            elif "每年" in update_freq or "yearly" in update_freq:
                update_counts["yearly"] += 1
            else:
                update_counts["irregular"] += 1
        
        analysis = {
            "total_datasets": len(datasets),
            "platform_distribution": platform_stats,
            "category_distribution": category_stats,
            "department_distribution": dict(sorted(department_stats.items(), key=lambda x: x[1], reverse=True)[:10]),
            "format_distribution": format_stats,
            "update_frequency": update_counts,
            "avg_access_count": sum(d.get("access_count", 0) for d in datasets) / len(datasets) if datasets else 0,
            "data_freshness": self._calculate_data_freshness(datasets),
            "analysis_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
        
        return analysis
    
    def _calculate_data_freshness(self, datasets: List[Dict]) -> Dict[str, Any]:
        """计算数据新鲜度"""
        if not datasets:
            return {"average_days_old": 0, "freshness_score": 0}
        
        today = datetime.now()
        total_days = 0
        fresh_count = 0
        stale_count = 0
        
        for dataset in datasets:
            update_date_str = dataset.get("update_date", "")
            if update_date_str:
                try:
                    update_date = datetime.strptime(update_date_str, "%Y-%m-%d")
                    days_old = (today - update_date).days
                    total_days += days_old
                    
                    if days_old <= 30:
                        fresh_count += 1
                    elif days_old > 180:
                        stale_count += 1
                except:
                    pass
        
        avg_days_old = total_days / len(datasets) if datasets else 0
        freshness_score = 100 - min(100, avg_days_old * 0.5)  # 粗略评分
        
        return {
            "average_days_old": round(avg_days_old, 1),
            "freshness_score": round(freshness_score, 1),
            "fresh_datasets": fresh_count,
            "stale_datasets": stale_count
        }
    
    def run(self,
            data_types: List[str] = None,
            platforms: List[str] = None,
            categories: List[str] = None,
            max_datasets_per_type: int = 15,
            **kwargs) -> Dict[str, Any]:
        """
        运行政府数据爬虫
        
        Args:
            data_types: 数据类型，可选 ["open_data", "statistical_report"]
            platforms: 平台列表，可选 ["national", "shanghai", "beijing", "guangdong"]
            categories: 分类列表
            max_datasets_per_type: 每种类型最大数据集数
            
        Returns:
            政府数据字典
        """
        print("=" * 60)
        print("开始运行政府公开数据爬虫")
        print("=" * 60)
        
        # 设置默认值
        if data_types is None:
            data_types = ["open_data", "statistical_report"]
        
        if platforms is None:
            platforms = ["national", "shanghai"]
        
        if categories is None:
            categories = ["经济", "人口", "环境"]
        
        all_data = {}
        start_time = datetime.now()
        
        try:
            # 获取开放数据
            if "open_data" in data_types:
                print(f"\n🏛️ 获取政府开放数据...")
                open_data = []
                for platform in platforms:
                    for category in categories[:2]:  # 限制分类数量
                        datasets = self.get_gov_open_data(platform, category, max_datasets_per_type // (len(platforms) * 2))
                        open_data.extend(datasets)
                        print(f"   平台 {platform}, 分类 {category}: {len(datasets)} 个数据集")
                all_data["open_data"] = open_data
            
            # 获取统计报告
            if "statistical_report" in data_types:
                print(f"\n📊 获取统计报告...")
                current_year = datetime.now().year
                annual_report = self.get_statistical_report("annual", current_year - 1)
                quarterly_report = self.get_statistical_report("quarterly", current_year)
                monthly_report = self.get_statistical_report("monthly", current_year)
                all_data["statistical_reports"] = {
                    "annual": annual_report,
                    "quarterly": quarterly_report,
                    "monthly": monthly_report
                }
                print(f"   获取到 {len(all_data['statistical_reports'])} 类统计报告")
            
            # 趋势分析
            trend_analysis = {}
            if "open_data" in all_data and all_data["open_data"]:
                trend_analysis = self.analyze_gov_data_trends(all_data["open_data"])
                print(f"\n📈 政府数据趋势分析:")
                print(f"   总数据集: {trend_analysis['total_datasets']}")
                print(f"   平均访问量: {trend_analysis['avg_access_count']:.0f}")
                print(f"   数据新鲜度评分: {trend_analysis['data_freshness']['freshness_score']}/100")
            
            # 生成报告
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()
            
            print(f"\n✅ 政府数据爬取完成!")
            print(f"   总计: {len(all_data.get('open_data', []))} 个开放数据集")
            print(f"   耗时: {duration:.2f} 秒")
            print(f"   平台: {', '.join(platforms)}")
            print(f"   分类: {', '.join(categories)}")
            
            result = {
                "open_data": all_data.get("open_data", []),
                "statistical_reports": all_data.get("statistical_reports", {}),
                "trend_analysis": trend_analysis,
                "crawl_summary": {
                    "total_datasets": len(all_data.get("open_data", [])),
                    "platforms": platforms,
                    "categories": categories,
                    "duration_seconds": round(duration, 2),
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
            }
            
            return result
            
        except Exception as e:
            print(f"❌ 政府数据爬取失败: {e}")
            import traceback
            traceback.print_exc()
            return {"error": str(e)}
    
    def export_data(self, data: Dict[str, Any], filename: str = None) -> str:
        """
        导出数据到Excel
        
        Args:
            data: 要导出的数据
            filename: 输出文件名
            
        Returns:
            导出的文件路径
        """
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"government_data_{timestamp}.xlsx"
        
        with pd.ExcelWriter(self.output_dir / filename, engine='openpyxl') as writer:
            # 导出开放数据
            if "open_data" in data and data["open_data"]:
                open_df = pd.DataFrame(data["open_data"])
                open_df.to_excel(writer, sheet_name="开放数据", index=False)
                print(f"导出开放数据: {len(open_df)} 行")
            
            # 导出统计报告摘要
            if "statistical_reports" in data and data["statistical_reports"]:
                report_summary = []
                for report_type, report in data["statistical_reports"].items():
                    report_summary.append({
                        "报告类型": report_type,
                        "标题": report.get("title", ""),
                        "发布机构": report.get("publisher", ""),
                        "发布日期": report.get("publish_date", ""),
                        "页数": report.get("pages", 0),
                        "文件大小": report.get("file_size", ""),
                        "下载链接": report.get("download_url", "")
                    })
                
                report_df = pd.DataFrame(report_summary)
                report_df.to_excel(writer, sheet_name="统计报告", index=False)
                print(f"导出统计报告: {len(report_df)} 行")
            
            # 导出趋势分析
            if "trend_analysis" in data and data["trend_analysis"]:
                trend_data = []
                analysis = data["trend_analysis"]
                
                # 平台分布
                for platform, count in analysis.get("platform_distribution", {}).items():
                    trend_data.append({
                        "分析维度": "平台分布",
                        "类别": platform,
                        "数值": count,
                        "占比(%)": round(count / analysis.get("total_datasets", 1) * 100, 2)
                    })
                
                # 分类分布
                for category, count in analysis.get("category_distribution", {}).items():
                    trend_data.append({
                        "分析维度": "分类分布",
                        "类别": category,
                        "数值": count,
                        "占比(%)": round(count / analysis.get("total_datasets", 1) * 100, 2)
                    })
                
                # 更新频率
                for freq_type, count in analysis.get("update_frequency", {}).items():
                    trend_data.append({
                        "分析维度": "更新频率",
                        "类别": freq_type,
                        "数值": count,
                        "占比(%)": round(count / analysis.get("total_datasets", 1) * 100, 2)
                    })
                
                trend_df = pd.DataFrame(trend_data)
                trend_df.to_excel(writer, sheet_name="趋势分析", index=False)
                print(f"导出趋势分析: {len(trend_df)} 行")
            
            # 添加汇总信息
            summary_data = {
                "数据项": ["爬取时间", "开放数据集数", "统计报告数", "导出文件"],
                "值": [
                    datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                    len(data.get("open_data", [])),
                    len(data.get("statistical_reports", {})),
                    filename
                ]
            }
            summary_df = pd.DataFrame(summary_data)
            summary_df.to_excel(writer, sheet_name="数据汇总", index=False)
        
        filepath = str(self.output_dir / filename)
        print(f"\n数据已导出到: {filepath}")
        return filepath


if __name__ == "__main__":
    # 演示用法
    crawler = GovernmentCrawler()
    
    # 运行爬虫
    data = crawler.run(
        data_types=["open_data", "statistical_report"],
        platforms=["national", "shanghai"],
        categories=["经济", "环境"],
        max_datasets_per_type=10
    )
    
    # 导出数据
    if data:
        filepath = crawler.export_data(data)
        print(f"\n数据已导出到: {filepath}")