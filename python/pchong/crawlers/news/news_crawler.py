#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
新闻资讯聚合爬虫 (CR002)
目标网站：新浪新闻、腾讯新闻、网易新闻、今日头条
"""

import time
import json
import re
from datetime import datetime
from typing import Dict, List, Optional
from urllib.parse import urljoin, urlparse

import requests
from bs4 import BeautifulSoup

from crawlers.base.base_crawler import BaseCrawler
from config.settings import specific_config

class NewsCrawler(BaseCrawler):
    """新闻资讯聚合爬虫"""
    
    def __init__(self):
        super().__init__("news_crawler")
        
        # 新闻源配置
        self.news_sources = {
            "sina": {
                "name": "新浪新闻",
                "base_url": "https://news.sina.com.cn",
                "hot_news_url": "https://news.sina.com.cn/hotnews/",
                "category_urls": {
                    "technology": "https://tech.sina.com.cn",
                    "politics": "https://news.sina.com.cn/china/",
                    "economics": "https://finance.sina.com.cn",
                    "sports": "https://sports.sina.com.cn"
                }
            },
            "tencent": {
                "name": "腾讯新闻",
                "base_url": "https://news.qq.com",
                "hot_news_url": "https://news.qq.com/hot/",
                "category_urls": {
                    "technology": "https://new.qq.com/ch/tech/",
                    "politics": "https://new.qq.com/ch/domestic/",
                    "economics": "https://new.qq.com/ch/finance/",
                    "sports": "https://new.qq.com/ch/sports/"
                }
            },
            "netease": {
                "name": "网易新闻",
                "base_url": "https://news.163.com",
                "hot_news_url": "https://news.163.com/hot/",
                "category_urls": {
                    "technology": "https://tech.163.com",
                    "politics": "https://news.163.com/domestic/",
                    "economics": "https://money.163.com",
                    "sports": "https://sports.163.com"
                }
            },
            "toutiao": {
                "name": "今日头条",
                "base_url": "https://www.toutiao.com",
                "hot_news_url": "https://www.toutiao.com/hot-event/hot-board/",
                "category_urls": {
                    "technology": "https://www.toutiao.com/ch/news_tech/",
                    "politics": "https://www.toutiao.com/ch/news_politics/",
                    "economics": "https://www.toutiao.com/ch/news_finance/",
                    "sports": "https://www.toutiao.com/ch/news_sports/"
                }
            }
        }
        
        # 爬取配置
        self.config = specific_config.NEWS.copy()
        
        self.logger.info("新闻资讯聚合爬虫初始化完成")
    
    def crawl(self, sources: List[str] = None, categories: List[str] = None, 
              max_articles: int = None, **kwargs) -> List[Dict]:
        """
        爬取新闻数据
        
        Args:
            sources: 新闻源列表
            categories: 新闻分类列表
            max_articles: 最大文章数
            
        Returns:
            新闻数据列表
        """
        if sources is None:
            sources = self.config["sources"]
        if categories is None:
            categories = self.config["categories"]
        if max_articles is None:
            max_articles = self.config["max_articles"]
        
        all_news = []
        
        for source_key in sources:
            if source_key not in self.news_sources:
                self.logger.warning(f"未知的新闻源: {source_key}")
                continue
            
            source = self.news_sources[source_key]
            self.logger.info(f"开始爬取 {source['name']}")
            
            # 爬取热点新闻
            hot_news = self._crawl_hot_news(source)
            if hot_news:
                all_news.extend(hot_news)
                self.logger.info(f"从 {source['name']} 爬取到 {len(hot_news)} 条热点新闻")
            
            # 爬取分类新闻
            for category in categories:
                if category in source["category_urls"]:
                    category_news = self._crawl_category_news(source, category)
                    if category_news:
                        all_news.extend(category_news)
                        self.logger.info(f"从 {source['name']} - {category} 爬取到 {len(category_news)} 条新闻")
                
                # 检查是否达到最大文章数
                if len(all_news) >= max_articles:
                    self.logger.info(f"已达到最大文章数: {max_articles}")
                    return all_news[:max_articles]
        
        return all_news[:max_articles]
    
    def _crawl_hot_news(self, source: Dict) -> List[Dict]:
        """爬取热点新闻"""
        try:
            response = self.make_request(source["hot_news_url"])
            if not response:
                return []
            
            # 根据不同网站解析热点新闻
            if source["name"] == "新浪新闻":
                return self._parse_sina_hot_news(response.text, source)
            elif source["name"] == "腾讯新闻":
                return self._parse_tencent_hot_news(response.text, source)
            elif source["name"] == "网易新闻":
                return self._parse_netease_hot_news(response.text, source)
            elif source["name"] == "今日头条":
                return self._parse_toutiao_hot_news(response.text, source)
            else:
                return []
                
        except Exception as e:
            self.logger.error(f"爬取热点新闻失败: {e}")
            return []
    
    def _crawl_category_news(self, source: Dict, category: str) -> List[Dict]:
        """爬取分类新闻"""
        try:
            url = source["category_urls"][category]
            response = self.make_request(url)
            if not response:
                return []
            
            soup = BeautifulSoup(response.text, 'html.parser')
            news_items = []
            
            # 通用解析逻辑（可根据具体网站调整）
            # 查找新闻链接
            news_links = []
            
            # 尝试不同的选择器
            selectors = [
                'a[href*="/article/"]',
                'a[href*="/news/"]',
                'a[href*="/doc/"]',
                'h3 a',
                '.title a',
                '.news-title a',
                'a.news-title'
            ]
            
            for selector in selectors:
                links = soup.select(selector)
                if links:
                    news_links.extend(links)
                    break
            
            # 去重
            seen_urls = set()
            unique_links = []
            for link in news_links:
                href = link.get('href', '')
                if href and href not in seen_urls:
                    seen_urls.add(href)
                    unique_links.append(link)
            
            # 限制数量
            max_items = 10
            for link in unique_links[:max_items]:
                href = link.get('href', '')
                if not href.startswith('http'):
                    href = urljoin(source["base_url"], href)
                
                # 获取新闻详情
                news_detail = self._get_news_detail(href, source["name"])
                if news_detail:
                    news_detail["category"] = category
                    news_items.append(news_detail)
                
                # 延迟避免请求过快
                self._random_delay(0.5, 1.5)
            
            return news_items
            
        except Exception as e:
            self.logger.error(f"爬取分类新闻失败: {e}")
            return []
    
    def _get_news_detail(self, url: str, source_name: str) -> Optional[Dict]:
        """获取新闻详情"""
        try:
            response = self.make_request(url)
            if not response:
                return None
            
            soup = BeautifulSoup(response.text, 'html.parser')
            
            # 提取标题
            title = ""
            title_selectors = ['h1', '.article-title', '.title', 'h1.title', '.news-title']
            for selector in title_selectors:
                title_elem = soup.select_one(selector)
                if title_elem:
                    title = title_elem.get_text(strip=True)
                    break
            
            # 提取内容
            content = ""
            content_selectors = ['.article-content', '.content', '.article', '.news-content', 'article']
            for selector in content_selectors:
                content_elem = soup.select_one(selector)
                if content_elem:
                    # 获取所有段落
                    paragraphs = content_elem.find_all('p')
                    if paragraphs:
                        content = '\n'.join([p.get_text(strip=True) for p in paragraphs if p.get_text(strip=True)])
                    else:
                        content = content_elem.get_text(strip=True)
                    break
            
            # 提取发布时间
            publish_time = ""
            time_selectors = ['.date', '.time', '.publish-time', '.pub-time', 'time']
            for selector in time_selectors:
                time_elem = soup.select_one(selector)
                if time_elem:
                    publish_time = time_elem.get_text(strip=True)
                    break
            
            # 提取作者
            author = ""
            author_selectors = ['.author', '.source', '.editor', '.writer']
            for selector in author_selectors:
                author_elem = soup.select_one(selector)
                if author_elem:
                    author = author_elem.get_text(strip=True)
                    break
            
            # 清理数据
            if content:
                content = re.sub(r'\s+', ' ', content).strip()
            
            # 创建新闻数据
            news_data = {
                "source": source_name,
                "title": title,
                "url": url,
                "content": content[:500] + "..." if len(content) > 500 else content,
                "content_length": len(content),
                "publish_time": publish_time,
                "author": author,
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                "has_content": bool(content)
            }
            
            return news_data
            
        except Exception as e:
            self.logger.error(f"获取新闻详情失败 {url}: {e}")
            return None
    
    def _parse_sina_hot_news(self, html: str, source: Dict) -> List[Dict]:
        """解析新浪热点新闻"""
        soup = BeautifulSoup(html, 'html.parser')
        news_items = []
        
        # 新浪热点新闻通常在一个表格中
        hot_news_table = soup.find('table', {'class': 'hot-news'})
        if not hot_news_table:
            # 尝试其他选择器
            hot_news_table = soup.find('div', {'class': 'hot-news'})
        
        if hot_news_table:
            links = hot_news_table.find_all('a', href=True)
            for link in links[:10]:  # 限制前10条
                href = link.get('href')
                title = link.get_text(strip=True)
                
                if href and title and not href.startswith('javascript'):
                    if not href.startswith('http'):
                        href = urljoin(source["base_url"], href)
                    
                    news_data = self._get_news_detail(href, source["name"])
                    if news_data:
                        news_items.append(news_data)
                    
                    self._random_delay(0.5, 1.5)
        
        return news_items
    
    def _parse_tencent_hot_news(self, html: str, source: Dict) -> List[Dict]:
        """解析腾讯热点新闻"""
        soup = BeautifulSoup(html, 'html.parser')
        news_items = []
        
        # 腾讯热点新闻选择器
        selectors = ['.hot-news-list', '.hot-list', '.news-list', '.list']
        
        for selector in selectors:
            news_list = soup.select(selector)
            if news_list:
                for news_elem in news_list[:10]:  # 限制前10条
                    links = news_elem.find_all('a', href=True)
                    for link in links:
                        href = link.get('href')
                        title = link.get_text(strip=True)
                        
                        if href and title:
                            if not href.startswith('http'):
                                href = urljoin(source["base_url"], href)
                            
                            news_data = self._get_news_detail(href, source["name"])
                            if news_data:
                                news_items.append(news_data)
                            
                            self._random_delay(0.5, 1.5)
                            break
                break
        
        return news_items
    
    def _parse_netease_hot_news(self, html: str, source: Dict) -> List[Dict]:
        """解析网易热点新闻"""
        soup = BeautifulSoup(html, 'html.parser')
        news_items = []
        
        # 网易热点新闻通常在一个列表中
        news_list = soup.find_all('div', {'class': 'data_row'})
        if not news_list:
            news_list = soup.find_all('li', {'class': 'news-item'})
        
        for news_elem in news_list[:10]:  # 限制前10条
            link = news_elem.find('a', href=True)
            if link:
                href = link.get('href')
                title = link.get_text(strip=True)
                
                if href and title:
                    if not href.startswith('http'):
                        href = urljoin(source["base_url"], href)
                    
                    news_data = self._get_news_detail(href, source["name"])
                    if news_data:
                        news_items.append(news_data)
                    
                    self._random_delay(0.5, 1.5)
        
        return news_items
    
    def _parse_toutiao_hot_news(self, html: str, source: Dict) -> List[Dict]:
        """解析今日头条热点新闻"""
        try:
            # 今日头条热点新闻通常是JSON格式
            data = json.loads(html)
            news_items = []
            
            if 'data' in data:
                for item in data['data'][:10]:  # 限制前10条
                    if 'Title' in item and 'Url' in item:
                        title = item['Title']
                        url = item['Url']
                        
                        # 今日头条的URL可能需要处理
                        if not url.startswith('http'):
                            url = f"https://www.toutiao.com{url}"
                        
                        news_data = {
                            "source": source["name"],
                            "title": title,
                            "url": url,
                            "content": item.get('Abstract', ''),
                            "publish_time": item.get('PublishTime', ''),
                            "hot_value": item.get('HotValue', 0),
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "has_content": False
                        }
                        
                        news_items.append(news_data)
            
            return news_items
            
        except json.JSONDecodeError:
            # 如果不是JSON，尝试HTML解析
            soup = BeautifulSoup(html, 'html.parser')
            news_items = []
            
            # 简单的链接提取
            links = soup.find_all('a', href=True)
            for link in links[:10]:
                href = link.get('href')
                title = link.get_text(strip=True)
                
                if href and title and '/article/' in href:
                    if not href.startswith('http'):
                        href = urljoin(source["base_url"], href)
                    
                    news_data = self._get_news_detail(href, source["name"])
                    if news_data:
                        news_items.append(news_data)
                    
                    self._random_delay(0.5, 1.5)
            
            return news_items


def demo_news_crawler():
    """演示新闻爬虫"""
    print("开始演示新闻资讯聚合爬虫...")
    
    # 创建爬虫实例
    crawler = NewsCrawler()
    
    # 运行爬虫（只爬取少量数据用于演示）
    try:
        news_data = crawler.run(
            sources=["sina", "tencent"],  # 只爬取新浪和腾讯
            categories=["technology"],     # 只爬取科技类
            max_articles=5                 # 只爬取5篇文章
        )
        
        print(f"\n成功爬取到 {len(news_data)} 条新闻:")
        for i, news in enumerate(news_data, 1):
            print(f"\n{i}. {news['source']} - {news['title']}")
            print(f"   链接: {news['url']}")
            if news.get('content'):
                print(f"   内容摘要: {news['content'][:100]}...")
            if news.get('publish_time'):
                print(f"   发布时间: {news['publish_time']}")
        
        # 保存数据到Excel（需要Excel导出模块）
        print("\n新闻数据已保存到JSON文件")
        
    except Exception as e:
        print(f"爬虫运行失败: {e}")


if __name__ == "__main__":
    demo_news_crawler()