#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
全局配置文件
"""

import os
from pathlib import Path

# 项目根目录
PROJECT_ROOT = Path(__file__).parent.parent

# 数据目录
DATA_DIR = PROJECT_ROOT / "data"
RAW_DATA_DIR = DATA_DIR / "raw"
PROCESSED_DATA_DIR = DATA_DIR / "processed"
EXCEL_DIR = DATA_DIR / "excel"

# 创建必要的目录
for directory in [DATA_DIR, RAW_DATA_DIR, PROCESSED_DATA_DIR, EXCEL_DIR]:
    directory.mkdir(exist_ok=True)

# 爬虫配置
class CrawlerConfig:
    # 请求配置
    REQUEST_TIMEOUT = 30
    MAX_RETRIES = 3
    RETRY_DELAY = 2  # 秒
    
    # 并发配置
    MAX_CONCURRENT_REQUESTS = 5
    DOWNLOAD_DELAY = 1  # 秒，避免请求过快
    
    # 用户代理
    USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    
    # 默认请求头
    DEFAULT_HEADERS = {
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
        "Accept-Encoding": "gzip, deflate, br",
        "Connection": "keep-alive",
        "Upgrade-Insecure-Requests": "1",
    }

# 日志配置
class LogConfig:
    LOG_DIR = PROJECT_ROOT / "logs"
    LOG_DIR.mkdir(exist_ok=True)
    
    LOG_LEVEL = "INFO"
    LOG_FORMAT = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    LOG_FILE = LOG_DIR / "crawler.log"
    
    # 每个爬虫单独日志
    CRAWLER_LOG_FORMAT = "%(asctime)s - %(levelname)s - %(message)s"

# 数据库配置
class DatabaseConfig:
    # SQLite配置
    SQLITE_DB_PATH = DATA_DIR / "crawler_data.db"
    
    # MongoDB配置（可选）
    MONGODB_URI = "mongodb://localhost:27017/"
    MONGODB_DB_NAME = "crawler_db"
    
    # PostgreSQL配置（可选）
    POSTGRES_HOST = "localhost"
    POSTGRES_PORT = 5432
    POSTGRES_DB = "crawler_data"
    POSTGRES_USER = "postgres"
    POSTGRES_PASSWORD = "password"

# 代理配置
class ProxyConfig:
    # 是否使用代理
    USE_PROXY = False
    
    # 代理服务器地址
    PROXY_SERVER = "http://your-proxy-server:port"
    
    # 代理认证
    PROXY_USER = None
    PROXY_PASSWORD = None
    
    # 代理轮换间隔（请求次数）
    PROXY_ROTATION_INTERVAL = 100

# Selenium配置
class SeleniumConfig:
    # 是否使用无头模式
    HEADLESS = True
    
    # 浏览器驱动程序路径
    CHROME_DRIVER_PATH = None  # 自动检测
    
    # 浏览器窗口大小
    WINDOW_SIZE = "1920,1080"
    
    # 隐式等待时间
    IMPLICIT_WAIT = 10
    
    # 显式等待超时时间
    EXPLICIT_WAIT_TIMEOUT = 30

# 各爬虫特定配置
class CrawlerSpecificConfig:
    # 电商爬虫配置
    ECOMMERCE = {
        "max_pages": 10,
        "products_per_page": 20,
        "category": "electronics",
        "price_range": {"min": 0, "max": 10000}
    }
    
    # 新闻爬虫配置
    NEWS = {
        "max_articles": 50,
        "sources": ["sina", "tencent", "netease", "toutiao"],
        "categories": ["technology", "politics", "economics", "sports"]
    }
    
    # 社交媒体配置
    SOCIAL_MEDIA = {
        "max_posts": 100,
        "platforms": ["weibo", "zhihu", "xiaohongshu", "douyin"],
        "keywords": ["technology", "lifestyle", "education"]
    }
    
    # 招聘网站配置
    JOB = {
        "max_jobs": 100,
        "positions": ["software engineer", "data scientist", "product manager"],
        "locations": ["北京", "上海", "深圳", "杭州"]
    }
    
    # 房地产配置
    REAL_ESTATE = {
        "max_properties": 100,
        "property_types": ["apartment", "house", "commercial"],
        "price_range": {"min": 1000000, "max": 10000000}
    }
    
    # 金融数据配置
    FINANCE = {
        "stocks": ["AAPL", "GOOGL", "MSFT", "TSLA"],
        "update_interval": 300,  # 秒
        "history_days": 30
    }
    
    # 学术论文配置
    ACADEMIC = {
        "max_papers": 50,
        "keywords": ["machine learning", "artificial intelligence", "data science"],
        "years": [2023, 2024]
    }
    
    # 旅游网站配置
    TRAVEL = {
        "max_hotels": 50,
        "check_in": "2024-12-01",
        "check_out": "2024-12-07",
        "locations": ["北京", "上海", "广州", "杭州"]
    }
    
    # 视频平台配置
    VIDEO = {
        "max_videos": 50,
        "categories": ["technology", "education", "entertainment"],
        "sort_by": "view_count"
    }
    
    # 政府数据配置
    GOVERNMENT = {
        "max_datasets": 50,
        "categories": ["economy", "population", "environment", "education"],
        "formats": ["csv", "json", "excel"]
    }

# 导出配置实例
crawler_config = CrawlerConfig()
log_config = LogConfig()
db_config = DatabaseConfig()
proxy_config = ProxyConfig()
selenium_config = SeleniumConfig()
specific_config = CrawlerSpecificConfig()

if __name__ == "__main__":
    print("配置文件加载成功")
    print(f"项目根目录: {PROJECT_ROOT}")
    print(f"数据目录: {DATA_DIR}")
    print(f"Excel输出目录: {EXCEL_DIR}")