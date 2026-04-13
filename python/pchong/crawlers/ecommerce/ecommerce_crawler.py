#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
电商价格监控爬虫 (CR001)
目标网站：淘宝、京东、拼多多、亚马逊（中国）
注意：由于电商网站反爬虫严格，此爬虫使用模拟浏览器和API接口
"""

import time
import re
import json
import random
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from urllib.parse import urlencode, quote, urljoin

import requests
from bs4 import BeautifulSoup
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, NoSuchElementException

from crawlers.base.base_crawler import BaseCrawler
from config.settings import specific_config, selenium_config

class EcommerceCrawler(BaseCrawler):
    """电商价格监控爬虫"""
    
    def __init__(self):
        super().__init__("ecommerce_crawler")
        
        # 电商平台配置
        self.platforms = {
            "taobao": {
                "name": "淘宝",
                "base_url": "https://s.taobao.com",
                "search_url": "https://s.taobao.com/search",
                "api_url": "https://api.m.taobao.com",
                "use_selenium": True,  # 需要Selenium处理动态内容
                "search_params": {
                    "q": "",  # 搜索关键词
                    "s": 0,   # 起始位置
                    "sort": "sale-desc"  # 按销量排序
                }
            },
            "jd": {
                "name": "京东",
                "base_url": "https://search.jd.com",
                "search_url": "https://search.jd.com/Search",
                "api_url": "https://api.m.jd.com",
                "use_selenium": True,
                "search_params": {
                    "keyword": "",
                    "enc": "utf-8",
                    "wq": "",
                    "psort": 3  # 按销量排序
                }
            },
            "pdd": {
                "name": "拼多多",
                "base_url": "https://mobile.yangkeduo.com",
                "search_url": "https://mobile.yangkeduo.com/search_result.html",
                "api_url": "https://api.pinduoduo.com",
                "use_selenium": True,
                "search_params": {
                    "search_key": "",
                    "page": 1
                }
            },
            "amazon_cn": {
                "name": "亚马逊中国",
                "base_url": "https://www.amazon.cn",
                "search_url": "https://www.amazon.cn/s",
                "use_selenium": True,
                "search_params": {
                    "k": "",
                    "ref": "nb_sb_noss"
                }
            }
        }
        
        # 爬取配置
        self.config = specific_config.ECOMMERCE.copy()
        
        # Selenium驱动
        self.driver = None
        
        # 商品分类映射
        self.categories = {
            "electronics": "电子产品",
            "clothing": "服装服饰", 
            "home": "家居用品",
            "books": "图书音像",
            "sports": "运动户外",
            "beauty": "美妆护肤",
            "food": "食品饮料",
            "other": "其他"
        }
        
        self.logger.info("电商价格监控爬虫初始化完成")
    
    def _init_selenium_driver(self):
        """初始化Selenium WebDriver"""
        if self.driver is not None:
            return self.driver
        
        try:
            chrome_options = Options()
            
            if selenium_config.HEADLESS:
                chrome_options.add_argument("--headless")
            
            chrome_options.add_argument("--no-sandbox")
            chrome_options.add_argument("--disable-dev-shm-usage")
            chrome_options.add_argument(f"--window-size={selenium_config.WINDOW_SIZE}")
            chrome_options.add_argument("--disable-blink-features=AutomationControlled")
            chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
            chrome_options.add_experimental_option('useAutomationExtension', False)
            
            # 添加随机User-Agent
            user_agent = self._get_random_user_agent()
            chrome_options.add_argument(f'user-agent={user_agent}')
            
            # 尝试自动获取ChromeDriver
            try:
                from selenium.webdriver.chrome.service import Service
                from webdriver_manager.chrome import ChromeDriverManager
                
                service = Service(ChromeDriverManager().install())
                self.driver = webdriver.Chrome(service=service, options=chrome_options)
                
            except ImportError:
                # 如果webdriver_manager不可用，使用系统ChromeDriver
                self.driver = webdriver.Chrome(options=chrome_options)
            
            # 执行JavaScript来隐藏自动化特征
            self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
            
            self.logger.info("Selenium WebDriver初始化成功")
            return self.driver
            
        except Exception as e:
            self.logger.error(f"Selenium WebDriver初始化失败: {e}")
            return None
    
    def _close_selenium_driver(self):
        """关闭Selenium WebDriver"""
        if self.driver:
            try:
                self.driver.quit()
                self.driver = None
                self.logger.info("Selenium WebDriver已关闭")
            except Exception as e:
                self.logger.error(f"关闭Selenium WebDriver失败: {e}")
    
    def crawl(self, platforms: List[str] = None, category: str = None, 
              max_pages: int = None, products_per_page: int = None,
              price_range: Dict = None, **kwargs) -> List[Dict]:
        """
        爬取电商商品数据
        
        Args:
            platforms: 电商平台列表
            category: 商品分类
            max_pages: 最大爬取页数
            products_per_page: 每页商品数
            price_range: 价格范围
            
        Returns:
            商品数据列表
        """
        if platforms is None:
            platforms = ["taobao", "jd", "pdd"]  # 默认爬取前三个平台
        
        if category is None:
            category = self.config["category"]
        
        if max_pages is None:
            max_pages = self.config["max_pages"]
        
        if products_per_page is None:
            products_per_page = self.config["products_per_page"]
        
        if price_range is None:
            price_range = self.config["price_range"]
        
        all_products = []
        
        for platform_key in platforms:
            if platform_key not in self.platforms:
                self.logger.warning(f"未知的电商平台: {platform_key}")
                continue
            
            platform = self.platforms[platform_key]
            self.logger.info(f"开始爬取 {platform['name']} - {category}")
            
            try:
                # 根据不同平台使用不同的爬取方法
                if platform["use_selenium"]:
                    products = self._crawl_with_selenium(platform, category, max_pages, products_per_page)
                else:
                    products = self._crawl_with_api(platform, category, max_pages, products_per_page)
                
                # 过滤价格范围
                if products:
                    filtered_products = self._filter_by_price(products, price_range)
                    all_products.extend(filtered_products)
                    self.logger.info(f"从 {platform['name']} 爬取到 {len(filtered_products)} 个商品")
                
            except Exception as e:
                self.logger.error(f"爬取 {platform['name']} 失败: {e}")
                continue
            
            # 平台间延迟
            self._random_delay(3, 5)
        
        # 关闭Selenium驱动
        self._close_selenium_driver()
        
        return all_products
    
    def _crawl_with_selenium(self, platform: Dict, category: str, 
                           max_pages: int, products_per_page: int) -> List[Dict]:
        """使用Selenium爬取动态页面"""
        driver = self._init_selenium_driver()
        if not driver:
            return []
        
        products = []
        
        try:
            # 构建搜索URL
            search_url = self._build_search_url(platform, category, 1)
            
            self.logger.info(f"访问搜索页面: {search_url}")
            driver.get(search_url)
            
            # 等待页面加载
            time.sleep(random.uniform(2, 4))
            
            # 模拟人类行为：滚动页面
            self._simulate_human_scroll(driver)
            
            # 解析商品列表
            page_products = self._parse_product_page(driver, platform)
            
            if page_products:
                products.extend(page_products)
                self.logger.info(f"第1页爬取到 {len(page_products)} 个商品")
            
            # 如果需要爬取多页
            if max_pages > 1:
                for page in range(2, max_pages + 1):
                    try:
                        # 构建下一页URL
                        next_page_url = self._build_search_url(platform, category, page)
                        
                        self.logger.info(f"访问第{page}页: {next_page_url}")
                        driver.get(next_page_url)
                        
                        # 等待页面加载
                        time.sleep(random.uniform(2, 4))
                        self._simulate_human_scroll(driver)
                        
                        # 解析商品
                        page_products = self._parse_product_page(driver, platform)
                        
                        if page_products:
                            products.extend(page_products)
                            self.logger.info(f"第{page}页爬取到 {len(page_products)} 个商品")
                        
                        # 页间延迟
                        time.sleep(random.uniform(1, 3))
                        
                    except Exception as e:
                        self.logger.error(f"爬取第{page}页失败: {e}")
                        break
            
            return products
            
        except Exception as e:
            self.logger.error(f"Selenium爬取失败: {e}")
            return []
    
    def _crawl_with_api(self, platform: Dict, category: str, 
                       max_pages: int, products_per_page: int) -> List[Dict]:
        """使用API接口爬取数据（备用方法）"""
        # 电商平台的API通常需要认证，这里实现基础版本
        products = []
        
        try:
            # 构建API请求
            api_params = {
                "keyword": category,
                "page": 1,
                "page_size": products_per_page,
                "sort": "sales_desc"
            }
            
            # 根据不同平台构建不同的API请求
            if platform["name"] == "淘宝":
                # 淘宝API需要特殊处理
                api_url = f"{platform['api_url']}/rest/api3.do"
                response = self.make_request(api_url, params=api_params)
                
            elif platform["name"] == "京东":
                # 京东API
                api_url = f"{platform['api_url']}/api"
                response = self.make_request(api_url, params=api_params)
            
            else:
                # 其他平台使用搜索页面
                return self._crawl_with_selenium(platform, category, 1, products_per_page)
            
            if response and response.status_code == 200:
                try:
                    data = response.json()
                    products = self._parse_api_response(data, platform)
                except:
                    # 如果API返回的不是JSON，尝试解析HTML
                    products = self._parse_html_response(response.text, platform)
            
            return products
            
        except Exception as e:
            self.logger.error(f"API爬取失败: {e}")
            return []
    
    def _build_search_url(self, platform: Dict, keyword: str, page: int) -> str:
        """构建搜索URL"""
        search_params = platform["search_params"].copy()
        
        # 根据平台设置关键词
        if platform["name"] == "淘宝":
            search_params["q"] = keyword
            search_params["s"] = (page - 1) * 44  # 淘宝每页44个商品
        elif platform["name"] == "京东":
            search_params["keyword"] = keyword
            search_params["page"] = page
        elif platform["name"] == "拼多多":
            search_params["search_key"] = keyword
            search_params["page"] = page
        elif platform["name"] == "亚马逊中国":
            search_params["k"] = keyword
            search_params["page"] = page
        
        # 构建URL
        if "search_url" in platform:
            url = platform["search_url"]
            if search_params:
                query_string = urlencode(search_params, doseq=True, safe=':/')
                url = f"{url}?{query_string}"
        
        return url
    
    def _simulate_human_scroll(self, driver):
        """模拟人类滚动行为"""
        try:
            # 随机滚动次数
            scroll_times = random.randint(3, 6)
            
            for i in range(scroll_times):
                # 随机滚动距离
                scroll_height = random.randint(300, 800)
                driver.execute_script(f"window.scrollBy(0, {scroll_height});")
                
                # 随机停留时间
                time.sleep(random.uniform(0.5, 1.5))
                
        except Exception as e:
            self.logger.warning(f"模拟滚动失败: {e}")
    
    def _parse_product_page(self, driver, platform: Dict) -> List[Dict]:
        """解析商品页面"""
        products = []
        
        try:
            # 根据不同平台使用不同的解析逻辑
            if platform["name"] == "淘宝":
                products = self._parse_taobao_products(driver)
            elif platform["name"] == "京东":
                products = self._parse_jd_products(driver)
            elif platform["name"] == "拼多多":
                products = self._parse_pdd_products(driver)
            elif platform["name"] == "亚马逊中国":
                products = self._parse_amazon_products(driver)
            
            return products
            
        except Exception as e:
            self.logger.error(f"解析商品页面失败: {e}")
            return []
    
    def _parse_taobao_products(self, driver) -> List[Dict]:
        """解析淘宝商品"""
        products = []
        
        try:
            # 淘宝商品选择器
            product_selectors = [
                '.item.J_MouserOnverReq',
                '.grid-item',
                '.item',
                '.Card--doubleCardWrapper'
            ]
            
            for selector in product_selectors:
                product_elements = driver.find_elements(By.CSS_SELECTOR, selector)
                if product_elements:
                    break
            
            for element in product_elements[:20]:  # 限制前20个商品
                try:
                    product_data = {}
                    
                    # 获取商品信息
                    product_html = element.get_attribute('outerHTML')
                    soup = BeautifulSoup(product_html, 'html.parser')
                    
                    # 提取标题
                    title_elem = soup.select_one('.title, .J_ClickStat, a[title]')
                    if title_elem:
                        title = title_elem.get('title') or title_elem.get_text(strip=True)
                        product_data["title"] = title[:100]  # 限制长度
                    
                    # 提取价格
                    price_elem = soup.select_one('.price, .price strong, .g_price')
                    if price_elem:
                        price_text = price_elem.get_text(strip=True)
                        price = self._extract_price(price_text)
                        if price:
                            product_data["price"] = price
                    
                    # 提取销量
                    sales_elem = soup.select_one('.deal-cnt, .sales')
                    if sales_elem:
                        sales_text = sales_elem.get_text(strip=True)
                        sales = self._extract_sales(sales_text)
                        if sales:
                            product_data["sales"] = sales
                    
                    # 提取店铺
                    shop_elem = soup.select_one('.shopname, .shop')
                    if shop_elem:
                        shop_name = shop_elem.get_text(strip=True)
                        product_data["shop_name"] = shop_name
                    
                    # 提取链接
                    link_elem = soup.select_one('a[href*="item.taobao.com"], a[href*="detail.tmall.com"]')
                    if link_elem:
                        href = link_elem.get('href', '')
                        if href and not href.startswith('http'):
                            href = urljoin('https:', href)
                        product_data["url"] = href
                    
                    # 补充信息
                    if product_data.get("title"):
                        product_data.update({
                            "platform": "淘宝",
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "product_id": self._generate_product_id(product_data)
                        })
                        products.append(product_data)
                    
                except Exception as e:
                    self.logger.debug(f"解析单个商品失败: {e}")
                    continue
        
        except Exception as e:
            self.logger.error(f"解析淘宝商品失败: {e}")
        
        return products
    
    def _parse_jd_products(self, driver) -> List[Dict]:
        """解析京东商品"""
        products = []
        
        try:
            # 京东商品选择器
            product_selectors = [
                '.gl-item',
                '.goods-item',
                '.item'
            ]
            
            for selector in product_selectors:
                product_elements = driver.find_elements(By.CSS_SELECTOR, selector)
                if product_elements:
                    break
            
            for element in product_elements[:20]:  # 限制前20个商品
                try:
                    product_data = {}
                    
                    # 获取商品信息
                    product_html = element.get_attribute('outerHTML')
                    soup = BeautifulSoup(product_html, 'html.parser')
                    
                    # 提取标题
                    title_elem = soup.select_one('.p-name em, .sku-name')
                    if title_elem:
                        title = title_elem.get_text(strip=True)
                        product_data["title"] = title[:100]
                    
                    # 提取价格
                    price_elem = soup.select_one('.p-price i, .J_price')
                    if price_elem:
                        price_text = price_elem.get_text(strip=True)
                        price = self._extract_price(price_text)
                        if price:
                            product_data["price"] = price
                    
                    # 提取评价数
                    comment_elem = soup.select_one('.p-commit a')
                    if comment_elem:
                        comment_text = comment_elem.get_text(strip=True)
                        comments = self._extract_sales(comment_text)
                        if comments:
                            product_data["comments"] = comments
                    
                    # 提取店铺
                    shop_elem = soup.select_one('.p-shop span a, .J_im_icon')
                    if shop_elem:
                        shop_name = shop_elem.get_text(strip=True)
                        product_data["shop_name"] = shop_name
                    
                    # 提取链接
                    link_elem = soup.select_one('a[href*="item.jd.com"]')
                    if link_elem:
                        href = link_elem.get('href', '')
                        if href and not href.startswith('http'):
                            href = urljoin('https://', href)
                        product_data["url"] = href
                    
                    # 补充信息
                    if product_data.get("title"):
                        product_data.update({
                            "platform": "京东",
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "product_id": self._generate_product_id(product_data)
                        })
                        products.append(product_data)
                    
                except Exception as e:
                    self.logger.debug(f"解析单个商品失败: {e}")
                    continue
        
        except Exception as e:
            self.logger.error(f"解析京东商品失败: {e}")
        
        return products
    
    def _parse_pdd_products(self, driver) -> List[Dict]:
        """解析拼多多商品"""
        products = []
        
        try:
            # 拼多多商品选择器
            product_selectors = [
                '.goods-item',
                '.item',
                '.list-item'
            ]
            
            for selector in product_selectors:
                product_elements = driver.find_elements(By.CSS_SELECTOR, selector)
                if product_elements:
                    break
            
            for element in product_elements[:20]:  # 限制前20个商品
                try:
                    product_data = {}
                    
                    # 获取商品信息
                    product_html = element.get_attribute('outerHTML')
                    soup = BeautifulSoup(product_html, 'html.parser')
                    
                    # 提取标题
                    title_elem = soup.select_one('.goods-name, .title')
                    if title_elem:
                        title = title_elem.get_text(strip=True)
                        product_data["title"] = title[:100]
                    
                    # 提取价格
                    price_elem = soup.select_one('.price, .goods-price')
                    if price_elem:
                        price_text = price_elem.get_text(strip=True)
                        price = self._extract_price(price_text)
                        if price:
                            product_data["price"] = price
                    
                    # 提取已拼数量
                    sales_elem = soup.select_one('.sales, .sold-count')
                    if sales_elem:
                        sales_text = sales_elem.get_text(strip=True)
                        sales = self._extract_sales(sales_text)
                        if sales:
                            product_data["sales"] = sales
                    
                    # 提取链接
                    link_elem = soup.select_one('a[href*="mobile.yangkeduo.com"]')
                    if link_elem:
                        href = link_elem.get('href', '')
                        if href:
                            product_data["url"] = href
                    
                    # 补充信息
                    if product_data.get("title"):
                        product_data.update({
                            "platform": "拼多多",
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "product_id": self._generate_product_id(product_data)
                        })
                        products.append(product_data)
                    
                except Exception as e:
                    self.logger.debug(f"解析单个商品失败: {e}")
                    continue
        
        except Exception as e:
            self.logger.error(f"解析拼多多商品失败: {e}")
        
        return products
    
    def _parse_amazon_products(self, driver) -> List[Dict]:
        """解析亚马逊商品"""
        products = []
        
        try:
            # 亚马逊商品选择器
            product_selectors = [
                '.s-result-item',
                '.s-main-slot .s-result-item'
            ]
            
            for selector in product_selectors:
                product_elements = driver.find_elements(By.CSS_SELECTOR, selector)
                if product_elements:
                    break
            
            for element in product_elements[:20]:  # 限制前20个商品
                try:
                    product_data = {}
                    
                    # 获取商品信息
                    product_html = element.get_attribute('outerHTML')
                    soup = BeautifulSoup(product_html, 'html.parser')
                    
                    # 提取标题
                    title_elem = soup.select_one('h2 a span')
                    if title_elem:
                        title = title_elem.get_text(strip=True)
                        product_data["title"] = title[:100]
                    
                    # 提取价格
                    price_elem = soup.select_one('.a-price .a-offscreen, .a-price-whole')
                    if price_elem:
                        price_text = price_elem.get_text(strip=True)
                        price = self._extract_price(price_text)
                        if price:
                            product_data["price"] = price
                    
                    # 提取评价
                    rating_elem = soup.select_one('.a-icon-alt')
                    if rating_elem:
                        rating_text = rating_elem.get_text(strip=True)
                        rating = self._extract_rating(rating_text)
                        if rating:
                            product_data["rating"] = rating
                    
                    # 提取链接
                    link_elem = soup.select_one('h2 a')
                    if link_elem:
                        href = link_elem.get('href', '')
                        if href and not href.startswith('http'):
                            href = urljoin('https://www.amazon.cn', href)
                        product_data["url"] = href
                    
                    # 补充信息
                    if product_data.get("title"):
                        product_data.update({
                            "platform": "亚马逊中国",
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "product_id": self._generate_product_id(product_data)
                        })
                        products.append(product_data)
                    
                except Exception as e:
                    self.logger.debug(f"解析单个商品失败: {e}")
                    continue
        
        except Exception as e:
            self.logger.error(f"解析亚马逊商品失败: {e}")
        
        return products
    
    def _extract_price(self, price_text: str) -> Optional[float]:
        """从文本中提取价格"""
        try:
            # 移除非数字字符（除了小数点和负号）
            clean_text = re.sub(r'[^\d\.\-]', '', price_text)
            if clean_text:
                return float(clean_text)
        except:
            pass
        return None
    
    def _extract_sales(self, sales_text: str) -> Optional[int]:
        """从文本中提取销量"""
        try:
            # 提取数字
            numbers = re.findall(r'\d+', sales_text)
            if numbers:
                # 处理"万"单位
                if '万' in sales_text or 'w' in sales_text.lower():
                    return int(float(numbers[0]) * 10000)
                return int(numbers[0])
        except:
            pass
        return None
    
    def _extract_rating(self, rating_text: str) -> Optional[float]:
        """从文本中提取评分"""
        try:
            # 提取评分数字
            match = re.search(r'(\d+\.?\d*)', rating_text)
            if match:
                return float(match.group(1))
        except:
            pass
        return None
    
    def _filter_by_price(self, products: List[Dict], price_range: Dict) -> List[Dict]:
        """按价格范围过滤商品"""
        filtered = []
        min_price = price_range.get("min", 0)
        max_price = price_range.get("max", float('inf'))
        
        for product in products:
            price = product.get("price")
            if price is not None and min_price <= price <= max_price:
                filtered.append(product)
        
        return filtered
    
    def _generate_product_id(self, product_data: Dict) -> str:
        """生成商品唯一ID"""
        import hashlib
        
        # 使用平台、标题和价格生成哈希ID
        key = f"{product_data.get('platform', '')}_{product_data.get('title', '')}_{product_data.get('price', '')}"
        return hashlib.md5(key.encode('utf-8')).hexdigest()[:10]
    
    def _parse_api_response(self, data: Dict, platform: Dict) -> List[Dict]:
        """解析API响应"""
        products = []
        
        try:
            # 通用API响应解析
            if isinstance(data, dict):
                # 尝试不同的数据路径
                items = data.get('data', data.get('result', data.get('items', [])))
                
                if isinstance(items, list):
                    for item in items[:20]:  # 限制前20个
                        product = self._parse_api_item(item, platform)
                        if product:
                            products.append(product)
            
            return products
            
        except Exception as e:
            self.logger.error(f"解析API响应失败: {e}")
            return []
    
    def _parse_api_item(self, item: Dict, platform: Dict) -> Optional[Dict]:
        """解析API返回的单个商品项"""
        try:
            product = {
                "platform": platform["name"],
                "title": item.get('title', item.get('name', '')),
                "price": item.get('price', item.get('current_price')),
                "original_price": item.get('original_price'),
                "sales": item.get('sales', item.get('sale_count')),
                "shop_name": item.get('shop_name', item.get('seller')),
                "url": item.get('url', item.get('link')),
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            }
            
            # 生成商品ID
            product["product_id"] = self._generate_product_id(product)
            
            # 清理空值
            product = {k: v for k, v in product.items() if v is not None}
            
            return product if product.get("title") else None
            
        except Exception as e:
            self.logger.debug(f"解析API商品项失败: {e}")
            return None
    
    def _parse_html_response(self, html: str, platform: Dict) -> List[Dict]:
        """解析HTML响应"""
        products = []
        
        try:
            soup = BeautifulSoup(html, 'html.parser')
            
            # 通用HTML解析
            product_elements = soup.select('.product, .item, .goods')
            
            for element in product_elements[:20]:  # 限制前20个
                try:
                    product = {}
                    
                    # 提取标题
                    title_elem = element.select_one('.title, .name, h3')
                    if title_elem:
                        product["title"] = title_elem.get_text(strip=True)[:100]
                    
                    # 提取价格
                    price_elem = element.select_one('.price, .current-price')
                    if price_elem:
                        price_text = price_elem.get_text(strip=True)
                        product["price"] = self._extract_price(price_text)
                    
                    # 提取链接
                    link_elem = element.select_one('a[href]')
                    if link_elem:
                        href = link_elem.get('href', '')
                        if href and not href.startswith('http'):
                            href = urljoin(platform["base_url"], href)
                        product["url"] = href
                    
                    # 补充信息
                    if product.get("title"):
                        product.update({
                            "platform": platform["name"],
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                            "product_id": self._generate_product_id(product)
                        })
                        products.append(product)
                    
                except Exception as e:
                    self.logger.debug(f"解析HTML商品失败: {e}")
                    continue
            
            return products
            
        except Exception as e:
            self.logger.error(f"解析HTML响应失败: {e}")
            return []


def demo_ecommerce_crawler():
    """演示电商爬虫"""
    print("演示: 运行电商价格监控爬虫")
    print("-" * 50)
    
    crawler = EcommerceCrawler()
    
    # 注意：实际运行时需要安装Chrome和ChromeDriver
    # 这里使用模拟数据演示
    
    print("由于电商网站反爬虫严格，此演示使用模拟数据")
    print("实际使用时请确保已正确配置Selenium环境")
    
    # 创建模拟数据
    mock_products = [
        {
            "product_id": "abc123",
            "platform": "淘宝",
            "title": "iPhone 15 Pro Max 256GB 原色钛金属",
            "price": 8999.00,
            "original_price": 9999.00,
            "sales": 1500,
            "shop_name": "苹果官方旗舰店",
            "url": "https://item.taobao.com/item.htm?id=123456",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        },
        {
            "product_id": "def456",
            "platform": "京东",
            "title": "华为Mate 60 Pro 12GB+512GB 雅川青",
            "price": 6999.00,
            "sales": 2800,
            "shop_name": "华为官方旗舰店",
            "url": "https://item.jd.com/123456.html",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        },
        {
            "product_id": "ghi789",
            "platform": "拼多多",
            "title": "小米14 Ultra 徕卡光学镜头 16GB+1TB",
            "price": 6499.00,
            "original_price": 6999.00,
            "sales": 1200,
            "shop_name": "小米官方旗舰店",
            "url": "https://mobile.yangkeduo.com/goods.html?goods_id=123456",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
    ]
    
    print(f"\n模拟爬取到 {len(mock_products)} 个商品:")
    for i, product in enumerate(mock_products, 1):
        print(f"\n{i}. [{product['platform']}] {product['title']}")
        print(f"   价格: ¥{product['price']:.2f}")
        if product.get('original_price'):
            print(f"   原价: ¥{product['original_price']:.2f}")
        if product.get('sales'):
            print(f"   销量: {product['sales']}")
        print(f"   店铺: {product.get('shop_name', 'N/A')}")
    
    return mock_products


if __name__ == "__main__":
    demo_ecommerce_crawler()