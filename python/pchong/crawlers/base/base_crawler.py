#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
基础爬虫类 - 所有爬虫的基类
"""

import time
import random
import logging
from abc import ABC, abstractmethod
from typing import Dict, List, Any, Optional
from pathlib import Path
import json

import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from fake_useragent import UserAgent

from config.settings import (
    crawler_config, 
    log_config,
    proxy_config,
    PROJECT_ROOT
)

class BaseCrawler(ABC):
    """基础爬虫抽象类"""
    
    def __init__(self, name: str, base_url: str = None):
        """
        初始化爬虫
        
        Args:
            name: 爬虫名称
            base_url: 基础URL
        """
        self.name = name
        self.base_url = base_url
        
        # 设置日志
        self.logger = self._setup_logger()
        
        # 会话对象
        self.session = self._create_session()
        
        # 数据存储
        self.data = []
        self.stats = {
            "total_requests": 0,
            "successful_requests": 0,
            "failed_requests": 0,
            "total_items": 0,
            "start_time": None,
            "end_time": None
        }
        
        # User-Agent生成器
        self.ua = UserAgent()
        
        self.logger.info(f"爬虫 '{self.name}' 初始化完成")
    
    def _setup_logger(self) -> logging.Logger:
        """设置日志记录器"""
        logger = logging.getLogger(f"crawler.{self.name}")
        
        if not logger.handlers:
            logger.setLevel(getattr(logging, log_config.LOG_LEVEL))
            
            # 控制台处理器
            console_handler = logging.StreamHandler()
            console_handler.setLevel(logging.INFO)
            console_formatter = logging.Formatter(log_config.CRAWLER_LOG_FORMAT)
            console_handler.setFormatter(console_formatter)
            logger.addHandler(console_handler)
            
            # 文件处理器
            log_file = log_config.LOG_DIR / f"{self.name}.log"
            file_handler = logging.FileHandler(log_file, encoding='utf-8')
            file_handler.setLevel(logging.DEBUG)
            file_formatter = logging.Formatter(log_config.LOG_FORMAT)
            file_handler.setFormatter(file_formatter)
            logger.addHandler(file_handler)
        
        return logger
    
    def _create_session(self) -> requests.Session:
        """创建带重试机制的会话"""
        session = requests.Session()
        
        # 设置重试策略
        retry_strategy = Retry(
            total=crawler_config.MAX_RETRIES,
            backoff_factor=crawler_config.RETRY_DELAY,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["HEAD", "GET", "OPTIONS"]
        )
        
        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("http://", adapter)
        session.mount("https://", adapter)
        
        # 设置默认请求头
        session.headers.update(crawler_config.DEFAULT_HEADERS)
        
        return session
    
    def _get_random_user_agent(self) -> str:
        """获取随机User-Agent"""
        return self.ua.random
    
    def _random_delay(self, min_delay: float = None, max_delay: float = None):
        """随机延迟，避免请求过快"""
        if min_delay is None:
            min_delay = crawler_config.DOWNLOAD_DELAY * 0.5
        if max_delay is None:
            max_delay = crawler_config.DOWNLOAD_DELAY * 1.5
        
        delay = random.uniform(min_delay, max_delay)
        time.sleep(delay)
    
    def make_request(self, url: str, method: str = "GET", **kwargs) -> Optional[requests.Response]:
        """
        发送HTTP请求
        
        Args:
            url: 请求URL
            method: HTTP方法
            **kwargs: 其他请求参数
            
        Returns:
            Response对象或None
        """
        self.stats["total_requests"] += 1
        
        # 添加随机User-Agent
        headers = kwargs.get("headers", {})
        headers["User-Agent"] = self._get_random_user_agent()
        kwargs["headers"] = headers
        
        # 设置超时
        if "timeout" not in kwargs:
            kwargs["timeout"] = crawler_config.REQUEST_TIMEOUT
        
        try:
            self.logger.debug(f"发送请求: {method} {url}")
            response = self.session.request(method, url, **kwargs)
            
            # 检查响应状态
            response.raise_for_status()
            
            self.stats["successful_requests"] += 1
            self.logger.debug(f"请求成功: {response.status_code}")
            
            return response
            
        except requests.exceptions.RequestException as e:
            self.stats["failed_requests"] += 1
            self.logger.error(f"请求失败: {url} - {e}")
            return None
    
    def parse_response(self, response: requests.Response) -> Any:
        """
        解析响应内容（子类需要实现具体解析逻辑）
        
        Args:
            response: Response对象
            
        Returns:
            解析后的数据
        """
        # 默认实现，返回文本内容
        return response.text
    
    def save_data(self, data: List[Dict], filename: str = None):
        """
        保存数据到文件
        
        Args:
            data: 要保存的数据
            filename: 文件名（不包含路径）
        """
        if not data:
            self.logger.warning("没有数据需要保存")
            return
        
        if filename is None:
            filename = f"{self.name}_{int(time.time())}.json"
        
        # 确保数据目录存在
        raw_data_dir = PROJECT_ROOT / "data" / "raw"
        raw_data_dir.mkdir(exist_ok=True)
        
        filepath = raw_data_dir / filename
        
        try:
            with open(filepath, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=2)
            
            self.logger.info(f"数据已保存到: {filepath}")
            self.logger.info(f"保存了 {len(data)} 条记录")
            
        except Exception as e:
            self.logger.error(f"保存数据失败: {e}")
    
    def get_stats(self) -> Dict:
        """获取爬虫统计信息"""
        if self.stats["start_time"] and self.stats["end_time"]:
            duration = self.stats["end_time"] - self.stats["start_time"]
            self.stats["duration_seconds"] = duration
        else:
            self.stats["duration_seconds"] = None
        
        return self.stats
    
    def print_stats(self):
        """打印爬虫统计信息"""
        stats = self.get_stats()
        
        print(f"\n{'='*50}")
        print(f"爬虫 '{self.name}' 统计信息")
        print(f"{'='*50}")
        print(f"总请求数: {stats['total_requests']}")
        print(f"成功请求数: {stats['successful_requests']}")
        print(f"失败请求数: {stats['failed_requests']}")
        print(f"总数据项: {stats['total_items']}")
        
        if stats['duration_seconds']:
            print(f"运行时间: {stats['duration_seconds']:.2f} 秒")
            if stats['total_items'] > 0:
                items_per_second = stats['total_items'] / stats['duration_seconds']
                print(f"爬取速度: {items_per_second:.2f} 项/秒")
        
        success_rate = 0
        if stats['total_requests'] > 0:
            success_rate = stats['successful_requests'] / stats['total_requests'] * 100
        
        print(f"成功率: {success_rate:.1f}%")
        print(f"{'='*50}")
    
    @abstractmethod
    def crawl(self, **kwargs) -> List[Dict]:
        """
        执行爬取操作（抽象方法，子类必须实现）
        
        Returns:
            爬取到的数据列表
        """
        pass
    
    def run(self, **kwargs) -> List[Dict]:
        """
        运行爬虫
        
        Returns:
            爬取到的数据列表
        """
        self.logger.info(f"开始运行爬虫 '{self.name}'")
        self.stats["start_time"] = time.time()
        
        try:
            # 执行爬取
            self.data = self.crawl(**kwargs)
            self.stats["total_items"] = len(self.data)
            
            # 保存数据
            if self.data:
                self.save_data(self.data)
            
            self.logger.info(f"爬虫 '{self.name}' 运行完成，爬取了 {len(self.data)} 条数据")
            
        except Exception as e:
            self.logger.error(f"爬虫运行失败: {e}")
            raise
        
        finally:
            self.stats["end_time"] = time.time()
            self.print_stats()
        
        return self.data


class SimpleWebCrawler(BaseCrawler):
    """简单的网页爬虫实现示例"""
    
    def __init__(self, name: str, base_url: str):
        super().__init__(name, base_url)
        self.visited_urls = set()
    
    def crawl(self, max_pages: int = 10, **kwargs) -> List[Dict]:
        """
        爬取网页内容
        
        Args:
            max_pages: 最大爬取页面数
            
        Returns:
            网页数据列表
        """
        data = []
        urls_to_visit = [self.base_url]
        
        while urls_to_visit and len(data) < max_pages:
            url = urls_to_visit.pop(0)
            
            if url in self.visited_urls:
                continue
            
            self.logger.info(f"爬取页面: {url}")
            
            response = self.make_request(url)
            if not response:
                continue
            
            # 解析页面
            page_data = self.parse_page(response.text, url)
            if page_data:
                data.append(page_data)
            
            # 提取链接（简单示例）
            # 在实际应用中，这里应该实现链接提取逻辑
            
            self.visited_urls.add(url)
            self._random_delay()
        
        return data
    
    def parse_page(self, html: str, url: str) -> Dict:
        """
        解析单个页面
        
        Args:
            html: 页面HTML
            url: 页面URL
            
        Returns:
            页面数据字典
        """
        # 这是一个示例实现
        # 在实际应用中，应该使用BeautifulSoup等库进行解析
        
        return {
            "url": url,
            "title": f"页面标题 - {url}",
            "content_length": len(html),
            "timestamp": time.time()
        }


if __name__ == "__main__":
    # 测试基础爬虫
    crawler = SimpleWebCrawler("test_crawler", "https://httpbin.org/get")
    data = crawler.run(max_pages=3)
    print(f"爬取到的数据: {data}")