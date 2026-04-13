#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
旅游网站酒店价格爬虫 (CR008)
目标网站：携程、去哪儿、飞猪
功能：酒店价格监控、房型信息、用户评价、地理位置
"""

import time
import re
import json
import random
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
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
from config.settings import specific_config, selenium_config, proxy_config


class TravelCrawler(BaseCrawler):
    """旅游网站酒店价格爬虫"""
    
    def __init__(self):
        super().__init__("travel_crawler")
        
        # 旅游平台配置
        self.platforms = {
            "ctrip": {
                "name": "携程",
                "base_url": "https://hotels.ctrip.com",
                "search_url": "https://hotels.ctrip.com",
                "api_url": "https://m.ctrip.com",
                "use_selenium": True,  # 需要Selenium处理动态内容
                "search_params": {
                    "city": "",        # 城市
                    "checkIn": "",     # 入住日期
                    "checkOut": "",    # 离店日期
                    "cityId": 0,       # 城市ID
                    "adultNum": 2,     # 成人数
                    "childNum": 0      # 儿童数
                },
                "hotel_types": {
                    "hotel": "酒店",
                    "apartment": "公寓",
                    "inn": "客栈",
                    "resort": "度假村",
                    "boutique": "精品酒店"
                }
            },
            "qunar": {
                "name": "去哪儿",
                "base_url": "https://hotel.qunar.com",
                "search_url": "https://hotel.qunar.com",
                "api_url": "https://api.qunar.com",
                "use_selenium": True,
                "search_params": {
                    "city": "",
                    "checkin_date": "",
                    "checkout_date": "",
                    "q": "",
                    "ps": 20,           # 每页数量
                    "pn": 1             # 页码
                },
                "price_range": {
                    "low": 100,         # 最低价格
                    "high": 2000        # 最高价格
                }
            },
            "fliggy": {
                "name": "飞猪",
                "base_url": "https://hotels.fliggy.com",
                "search_url": "https://hotels.fliggy.com",
                "api_url": "https://acs.fliggy.com",
                "use_selenium": True,
                "search_params": {
                    "city": "",
                    "checkIn": "",
                    "checkOut": "",
                    "adultNum": 2,
                    "childNum": 0,
                    "keywords": ""
                },
                "sort_options": {
                    "default": "默认排序",
                    "price_low": "价格从低到高",
                    "price_high": "价格从高到低",
                    "score_high": "评分从高到低"
                }
            }
        }
        
        # 爬取配置
        self.config = specific_config.TRAVEL.copy()
        
        # Selenium驱动
        self.driver = None
        
        # 城市映射
        self.city_mapping = {
            "北京": "beijing",
            "上海": "shanghai", 
            "广州": "guangzhou",
            "杭州": "hangzhou",
            "成都": "chengdu",
            "西安": "xian",
            "南京": "nanjing",
            "深圳": "shenzhen",
            "重庆": "chongqing",
            "武汉": "wuhan"
        }
        
        # 酒店星级映射
        self.star_mapping = {
            "1": "经济型",
            "2": "舒适型", 
            "3": "高档型",
            "4": "豪华型",
            "5": "奢华型"
        }
        
        # 房型映射
        self.room_type_mapping = {
            "standard": "标准间",
            "deluxe": "豪华间",
            "suite": "套房",
            "family": "家庭房",
            "business": "商务房",
            "view": "景观房"
        }
        
        # 数据字段定义
        self.data_fields = [
            "hotel_id", "hotel_name", "platform", "city", "address",
            "star_rating", "user_rating", "review_count", "lowest_price",
            "original_price", "discount", "room_type", "room_name",
            "breakfast", "wifi", "parking", "check_in", "check_out",
            "latitude", "longitude", "phone", "facilities", "tags",
            "crawl_time", "source_url"
        ]
    
    def _setup_selenium_driver(self):
        """设置Selenium驱动"""
        try:
            from selenium.webdriver.chrome.options import Options
            chrome_options = Options()
            chrome_options.add_argument("--headless")  # 无头模式
            chrome_options.add_argument("--no-sandbox")
            chrome_options.add_argument("--disable-dev-shm-usage")
            chrome_options.add_argument("--disable-gpu")
            chrome_options.add_argument("--window-size=1920,1080")
            chrome_options.add_argument(f"user-agent={selenium_config.USER_AGENT}")
            
            # 添加代理
            if proxy_config.ENABLED and proxy_config.HTTP_PROXY:
                chrome_options.add_argument(f"--proxy-server={proxy_config.HTTP_PROXY}")
            
            self.driver = webdriver.Chrome(options=chrome_options)
            self.driver.set_page_load_timeout(selenium_config.PAGE_LOAD_TIMEOUT)
            
            self.logger.info("Selenium驱动初始化成功")
            return True
        except ImportError:
            self.logger.warning("Selenium未安装，使用模拟数据模式")
            return False
        except Exception as e:
            self.logger.error(f"Selenium驱动初始化失败: {e}")
            return False
    
    def _search_ctrip_hotels(self, city: str, check_in: str = None, check_out: str = None) -> List[Dict]:
        """搜索携程酒店"""
        hotels = []
        
        try:
            if not self.driver:
                if not self._setup_selenium_driver():
                    return hotels
            
            # 构建搜索URL
            city_code = self.city_mapping.get(city, "beijing")
            check_in_date = check_in or self.config.get("check_in", "2024-12-01")
            check_out_date = check_out or self.config.get("check_out", "2024-12-07")
            
            search_url = f"https://hotels.ctrip.com/hotel/{city_code}/#ctm_ref=hod_hp_sb_lst"
            
            self.logger.info(f"开始搜索携程酒店: {city}, 入住: {check_in_date}, 离店: {check_out_date}")
            self.driver.get(search_url)
            
            # 等待页面加载
            time.sleep(random.uniform(2, 4))
            
            # 模拟搜索操作
            try:
                # 输入入住日期
                checkin_input = self.driver.find_element(By.CSS_SELECTOR, "input[name='checkIn']")
                checkin_input.clear()
                checkin_input.send_keys(check_in_date)
                
                # 输入离店日期
                checkout_input = self.driver.find_element(By.CSS_SELECTOR, "input[name='checkOut']")
                checkout_input.clear()
                checkout_input.send_keys(check_out_date)
                
                # 点击搜索按钮
                search_btn = self.driver.find_element(By.CSS_SELECTOR, "button.search-btn")
                search_btn.click()
                
                time.sleep(random.uniform(3, 5))
                
            except Exception as e:
                self.logger.debug(f"搜索表单操作失败，继续使用默认页面: {e}")
            
            # 解析酒店列表
            hotel_elements = self.driver.find_elements(By.CSS_SELECTOR, "div.hotel-item")
            
            for i, hotel_element in enumerate(hotel_elements[:self.config.get("max_hotels", 10)]):
                try:
                    hotel_info = self._parse_ctrip_hotel(hotel_element)
                    if hotel_info:
                        hotel_info.update({
                            "platform": "携程",
                            "city": city,
                            "check_in": check_in_date,
                            "check_out": check_out_date,
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                        })
                        hotels.append(hotel_info)
                        self.logger.debug(f"成功解析携程酒店 {i+1}: {hotel_info.get('hotel_name', '未知')}")
                except Exception as e:
                    self.logger.error(f"解析携程酒店失败: {e}")
                
                # 随机延迟，避免请求过快
                if i % 5 == 0:
                    time.sleep(random.uniform(0.5, 1.5))
            
            self.logger.info(f"携程酒店搜索完成，找到 {len(hotels)} 家酒店")
            
        except Exception as e:
            self.logger.error(f"携程酒店搜索失败: {e}")
        
        return hotels
    
    def _parse_ctrip_hotel(self, hotel_element) -> Optional[Dict]:
        """解析携程酒店元素"""
        try:
            hotel_info = {}
            
            # 获取酒店名称
            name_element = hotel_element.find_element(By.CSS_SELECTOR, "h2.hotel-name a")
            hotel_info["hotel_name"] = name_element.text.strip()
            hotel_info["source_url"] = name_element.get_attribute("href")
            
            # 获取酒店ID
            if "hotel/" in hotel_info["source_url"]:
                match = re.search(r"hotel/(\d+)", hotel_info["source_url"])
                if match:
                    hotel_info["hotel_id"] = match.group(1)
            
            # 获取地址
            try:
                address_element = hotel_element.find_element(By.CSS_SELECTOR, "p.hotel-address")
                hotel_info["address"] = address_element.text.strip()
            except:
                hotel_info["address"] = ""
            
            # 获取评分
            try:
                rating_element = hotel_element.find_element(By.CSS_SELECTOR, "span.hotel-score")
                hotel_info["user_rating"] = float(rating_element.text.strip())
            except:
                hotel_info["user_rating"] = 0.0
            
            # 获取评论数量
            try:
                review_element = hotel_element.find_element(By.CSS_SELECTOR, "span.review-count")
                review_text = review_element.text.strip()
                match = re.search(r"(\d+)", review_text)
                if match:
                    hotel_info["review_count"] = int(match.group(1))
            except:
                hotel_info["review_count"] = 0
            
            # 获取价格
            try:
                price_element = hotel_element.find_element(By.CSS_SELECTOR, "span.price strong")
                price_text = price_element.text.strip()
                match = re.search(r"(\d+)", price_text.replace(",", ""))
                if match:
                    hotel_info["lowest_price"] = int(match.group(1))
            except:
                hotel_info["lowest_price"] = 0
            
            # 获取原价（如果有折扣）
            try:
                original_price_element = hotel_element.find_element(By.CSS_SELECTOR, "del.price")
                original_text = original_price_element.text.strip()
                match = re.search(r"(\d+)", original_text.replace(",", ""))
                if match:
                    hotel_info["original_price"] = int(match.group(1))
                    if hotel_info["lowest_price"] > 0 and hotel_info["original_price"] > hotel_info["lowest_price"]:
                        hotel_info["discount"] = round((1 - hotel_info["lowest_price"] / hotel_info["original_price"]) * 100, 1)
            except:
                hotel_info["original_price"] = hotel_info["lowest_price"]
            
            # 获取星级
            try:
                star_element = hotel_element.find_element(By.CSS_SELECTOR, "span.hotel-star")
                star_text = star_element.get_attribute("class") or star_element.text
                if "star5" in star_text or "五星" in star_text:
                    hotel_info["star_rating"] = 5
                elif "star4" in star_text or "四星" in star_text:
                    hotel_info["star_rating"] = 4
                elif "star3" in star_text or "三星" in star_text:
                    hotel_info["star_rating"] = 3
                elif "star2" in star_text or "二星" in star_text:
                    hotel_info["star_rating"] = 2
                else:
                    hotel_info["star_rating"] = 1
            except:
                hotel_info["star_rating"] = 0
            
            # 获取标签
            try:
                tags_elements = hotel_element.find_elements(By.CSS_SELECTOR, "span.hotel-tag")
                tags = [tag.text.strip() for tag in tags_elements]
                hotel_info["tags"] = ",".join(tags)
            except:
                hotel_info["tags"] = ""
            
            # 获取设施
            try:
                facilities_elements = hotel_element.find_elements(By.CSS_SELECTOR, "span.facility-icon")
                facilities = []
                for facility in facilities_elements:
                    title = facility.get_attribute("title") or ""
                    if title:
                        facilities.append(title)
                hotel_info["facilities"] = ",".join(facilities)
            except:
                hotel_info["facilities"] = ""
            
            return hotel_info
            
        except Exception as e:
            self.logger.error(f"解析携程酒店元素失败: {e}")
            return None
    
    def _search_qunar_hotels(self, city: str, check_in: str = None, check_out: str = None) -> List[Dict]:
        """搜索去哪儿酒店"""
        hotels = []
        
        try:
            if not self.driver:
                if not self._setup_selenium_driver():
                    return hotels
            
            # 构建搜索URL
            check_in_date = check_in or self.config.get("check_in", "2024-12-01")
            check_out_date = check_out or self.config.get("check_out", "2024-12-07")
            
            # 去哪儿酒店搜索页面
            search_url = f"https://hotel.qunar.com/city/{city}/?fromDate={check_in_date}&toDate={check_out_date}"
            
            self.logger.info(f"开始搜索去哪儿酒店: {city}, 入住: {check_in_date}, 离店: {check_out_date}")
            self.driver.get(search_url)
            
            # 等待页面加载
            time.sleep(random.uniform(2, 4))
            
            # 解析酒店列表
            hotel_elements = self.driver.find_elements(By.CSS_SELECTOR, "div.hotel-item")
            
            for i, hotel_element in enumerate(hotel_elements[:self.config.get("max_hotels", 10)]):
                try:
                    hotel_info = self._parse_qunar_hotel(hotel_element)
                    if hotel_info:
                        hotel_info.update({
                            "platform": "去哪儿",
                            "city": city,
                            "check_in": check_in_date,
                            "check_out": check_out_date,
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                        })
                        hotels.append(hotel_info)
                        self.logger.debug(f"成功解析去哪儿酒店 {i+1}: {hotel_info.get('hotel_name', '未知')}")
                except Exception as e:
                    self.logger.error(f"解析去哪儿酒店失败: {e}")
                
                # 随机延迟
                if i % 5 == 0:
                    time.sleep(random.uniform(0.5, 1.5))
            
            self.logger.info(f"去哪儿酒店搜索完成，找到 {len(hotels)} 家酒店")
            
        except Exception as e:
            self.logger.error(f"去哪儿酒店搜索失败: {e}")
        
        return hotels
    
    def _parse_qunar_hotel(self, hotel_element) -> Optional[Dict]:
        """解析去哪儿酒店元素"""
        try:
            hotel_info = {}
            
            # 获取酒店名称和链接
            try:
                name_element = hotel_element.find_element(By.CSS_SELECTOR, "h2.hotel-name a")
                hotel_info["hotel_name"] = name_element.text.strip()
                hotel_info["source_url"] = name_element.get_attribute("href")
            except:
                # 备用选择器
                name_element = hotel_element.find_element(By.CSS_SELECTOR, "a.hotel-name")
                hotel_info["hotel_name"] = name_element.text.strip()
                hotel_info["source_url"] = name_element.get_attribute("href")
            
            # 获取酒店ID
            if "/hotel/" in hotel_info["source_url"]:
                match = re.search(r"/hotel/(\d+)", hotel_info["source_url"])
                if match:
                    hotel_info["hotel_id"] = match.group(1)
            
            # 获取地址
            try:
                address_element = hotel_element.find_element(By.CSS_SELECTOR, "p.hotel-address")
                hotel_info["address"] = address_element.text.strip()
            except:
                hotel_info["address"] = ""
            
            # 获取评分
            try:
                rating_element = hotel_element.find_element(By.CSS_SELECTOR, "span.score")
                hotel_info["user_rating"] = float(rating_element.text.strip())
            except:
                try:
                    rating_element = hotel_element.find_element(By.CSS_SELECTOR, "span.rating")
                    hotel_info["user_rating"] = float(rating_element.text.strip())
                except:
                    hotel_info["user_rating"] = 0.0
            
            # 获取评论数量
            try:
                review_element = hotel_element.find_element(By.CSS_SELECTOR, "span.review-count")
                review_text = review_element.text.strip()
                match = re.search(r"(\d+)", review_text)
                if match:
                    hotel_info["review_count"] = int(match.group(1))
            except:
                hotel_info["review_count"] = 0
            
            # 获取价格
            try:
                price_element = hotel_element.find_element(By.CSS_SELECTOR, "span.price strong")
                price_text = price_element.text.strip()
                match = re.search(r"(\d+)", price_text.replace("¥", "").replace(",", ""))
                if match:
                    hotel_info["lowest_price"] = int(match.group(1))
            except:
                try:
                    price_element = hotel_element.find_element(By.CSS_SELECTOR, "b.price")
                    price_text = price_element.text.strip()
                    match = re.search(r"(\d+)", price_text.replace("¥", "").replace(",", ""))
                    if match:
                        hotel_info["lowest_price"] = int(match.group(1))
                except:
                    hotel_info["lowest_price"] = 0
            
            # 获取星级
            try:
                star_element = hotel_element.find_element(By.CSS_SELECTOR, "span.hotel-star")
                star_text = star_element.get_attribute("class") or star_element.text
                if "star5" in star_text or "五星" in star_text:
                    hotel_info["star_rating"] = 5
                elif "star4" in star_text or "四星" in star_text:
                    hotel_info["star_rating"] = 4
                elif "star3" in star_text or "三星" in star_text:
                    hotel_info["star_rating"] = 3
                elif "star2" in star_text or "二星" in star_text:
                    hotel_info["star_rating"] = 2
                else:
                    hotel_info["star_rating"] = 1
            except:
                hotel_info["star_rating"] = 0
            
            # 获取标签
            try:
                tags_elements = hotel_element.find_elements(By.CSS_SELECTOR, "span.tag")
                tags = [tag.text.strip() for tag in tags_elements]
                hotel_info["tags"] = ",".join(tags)
            except:
                hotel_info["tags"] = ""
            
            # 获取推荐理由
            try:
                recommend_element = hotel_element.find_element(By.CSS_SELECTOR, "p.recommend-reason")
                hotel_info["recommend_reason"] = recommend_element.text.strip()
            except:
                hotel_info["recommend_reason"] = ""
            
            return hotel_info
            
        except Exception as e:
            self.logger.error(f"解析去哪儿酒店元素失败: {e}")
            return None
    
    def _search_fliggy_hotels(self, city: str, check_in: str = None, check_out: str = None) -> List[Dict]:
        """搜索飞猪酒店"""
        hotels = []
        
        try:
            if not self.driver:
                if not self._setup_selenium_driver():
                    return hotels
            
            # 构建搜索URL
            check_in_date = check_in or self.config.get("check_in", "2024-12-01")
            check_out_date = check_out or self.config.get("check_out", "2024-12-07")
            
            # 飞猪酒店搜索
            city_encoded = quote(city)
            search_url = f"https://hotels.fliggy.com/?city={city_encoded}&checkIn={check_in_date}&checkOut={check_out_date}"
            
            self.logger.info(f"开始搜索飞猪酒店: {city}, 入住: {check_in_date}, 离店: {check_out_date}")
            self.driver.get(search_url)
            
            # 等待页面加载
            time.sleep(random.uniform(2, 4))
            
            # 解析酒店列表
            hotel_elements = self.driver.find_elements(By.CSS_SELECTOR, "div.hotel-item")
            
            for i, hotel_element in enumerate(hotel_elements[:self.config.get("max_hotels", 10)]):
                try:
                    hotel_info = self._parse_fliggy_hotel(hotel_element)
                    if hotel_info:
                        hotel_info.update({
                            "platform": "飞猪",
                            "city": city,
                            "check_in": check_in_date,
                            "check_out": check_out_date,
                            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                        })
                        hotels.append(hotel_info)
                        self.logger.debug(f"成功解析飞猪酒店 {i+1}: {hotel_info.get('hotel_name', '未知')}")
                except Exception as e:
                    self.logger.error(f"解析飞猪酒店失败: {e}")
                
                # 随机延迟
                if i % 5 == 0:
                    time.sleep(random.uniform(0.5, 1.5))
            
            self.logger.info(f"飞猪酒店搜索完成，找到 {len(hotels)} 家酒店")
            
        except Exception as e:
            self.logger.error(f"飞猪酒店搜索失败: {e}")
        
        return hotels
    
    def _parse_fliggy_hotel(self, hotel_element) -> Optional[Dict]:
        """解析飞猪酒店元素"""
        try:
            hotel_info = {}
            
            # 获取酒店名称和链接
            try:
                name_element = hotel_element.find_element(By.CSS_SELECTOR, "a.hotel-name")
                hotel_info["hotel_name"] = name_element.text.strip()
                hotel_info["source_url"] = name_element.get_attribute("href")
            except:
                hotel_info["hotel_name"] = ""
                hotel_info["source_url"] = ""
            
            # 获取酒店ID
            if "item/" in hotel_info["source_url"]:
                match = re.search(r"item/(\d+)", hotel_info["source_url"])
                if match:
                    hotel_info["hotel_id"] = match.group(1)
            
            # 获取地址
            try:
                address_element = hotel_element.find_element(By.CSS_SELECTOR, "p.hotel-address")
                hotel_info["address"] = address_element.text.strip()
            except:
                hotel_info["address"] = ""
            
            # 获取评分
            try:
                rating_element = hotel_element.find_element(By.CSS_SELECTOR, "span.score")
                hotel_info["user_rating"] = float(rating_element.text.strip())
            except:
                hotel_info["user_rating"] = 0.0
            
            # 获取价格
            try:
                price_element = hotel_element.find_element(By.CSS_SELECTOR, "span.price strong")
                price_text = price_element.text.strip()
                match = re.search(r"(\d+)", price_text.replace("¥", "").replace(",", ""))
                if match:
                    hotel_info["lowest_price"] = int(match.group(1))
            except:
                hotel_info["lowest_price"] = 0
            
            # 获取原价
            try:
                original_price_element = hotel_element.find_element(By.CSS_SELECTOR, "span.original-price")
                original_text = original_price_element.text.strip()
                match = re.search(r"(\d+)", original_text.replace("¥", "").replace(",", ""))
                if match:
                    hotel_info["original_price"] = int(match.group(1))
                    if hotel_info["lowest_price"] > 0 and hotel_info["original_price"] > hotel_info["lowest_price"]:
                        hotel_info["discount"] = round((1 - hotel_info["lowest_price"] / hotel_info["original_price"]) * 100, 1)
            except:
                hotel_info["original_price"] = hotel_info["lowest_price"]
            
            # 获取星级
            try:
                star_element = hotel_element.find_element(By.CSS_SELECTOR, "span.hotel-star")
                star_text = star_element.text or star_element.get_attribute("class")
                if "5星" in star_text or "五星" in star_text:
                    hotel_info["star_rating"] = 5
                elif "4星" in star_text or "四星" in star_text:
                    hotel_info["star_rating"] = 4
                elif "3星" in star_text or "三星" in star_text:
                    hotel_info["star_rating"] = 3
                elif "2星" in star_text or "二星" in star_text:
                    hotel_info["star_rating"] = 2
                else:
                    hotel_info["star_rating"] = 1
            except:
                hotel_info["star_rating"] = 0
            
            # 获取标签
            try:
                tags_elements = hotel_element.find_elements(By.CSS_SELECTOR, "span.tag")
                tags = [tag.text.strip() for tag in tags_elements]
                hotel_info["tags"] = ",".join(tags)
            except:
                hotel_info["tags"] = ""
            
            # 获取设施
            try:
                facilities_elements = hotel_element.find_elements(By.CSS_SELECTOR, "span.facility")
                facilities = [facility.text.strip() for facility in facilities_elements]
                hotel_info["facilities"] = ",".join(facilities)
            except:
                hotel_info["facilities"] = ""
            
            return hotel_info
            
        except Exception as e:
            self.logger.error(f"解析飞猪酒店元素失败: {e}")
            return None
    
    def run(self, **kwargs) -> Dict[str, Any]:
        """
        运行旅游网站酒店价格爬虫
        
        Args:
            **kwargs: 爬虫参数，可以包含:
                - city: 城市名称
                - check_in: 入住日期 (格式: YYYY-MM-DD)
                - check_out: 离店日期 (格式: YYYY-MM-DD)
                - platform: 平台名称 (ctrip/qunar/fliggy/all)
                - max_hotels: 最大酒店数量
            
        Returns:
            爬取数据
        """
        # 参数处理
        city = kwargs.get("city", self.config.get("locations", ["北京"])[0])
        check_in = kwargs.get("check_in", self.config.get("check_in", "2024-12-01"))
        check_out = kwargs.get("check_out", self.config.get("check_out", "2024-12-07"))
        platform = kwargs.get("platform", "all")
        max_hotels = kwargs.get("max_hotels", self.config.get("max_hotels", 50))
        
        # 记录开始时间
        start_time = time.time()
        self.logger.info(f"开始运行旅游网站酒店价格爬虫")
        self.logger.info(f"参数: 城市={city}, 入住={check_in}, 离店={check_out}, 平台={platform}")
        
        all_hotels = []
        stats = {
            "total_hotels": 0,
            "platforms_searched": 0,
            "cities_searched": 0,
            "success_rate": 0,
            "total_time": 0
        }
        
        try:
            # 根据平台选择搜索方式
            if platform in ["ctrip", "all"]:
                self.logger.info(f"开始搜索携程酒店...")
                ctrip_hotels = self._search_ctrip_hotels(city, check_in, check_out)
                all_hotels.extend(ctrip_hotels)
                stats["platforms_searched"] += 1
                self.logger.info(f"携程酒店搜索完成: {len(ctrip_hotels)} 家")
            
            if platform in ["qunar", "all"]:
                self.logger.info(f"开始搜索去哪儿酒店...")
                qunar_hotels = self._search_qunar_hotels(city, check_in, check_out)
                all_hotels.extend(qunar_hotels)
                stats["platforms_searched"] += 1
                self.logger.info(f"去哪儿酒店搜索完成: {len(qunar_hotels)} 家")
            
            if platform in ["fliggy", "all"]:
                self.logger.info(f"开始搜索飞猪酒店...")
                fliggy_hotels = self._search_fliggy_hotels(city, check_in, check_out)
                all_hotels.extend(fliggy_hotels)
                stats["platforms_searched"] += 1
                self.logger.info(f"飞猪酒店搜索完成: {len(fliggy_hotels)} 家")
            
            # 限制最大数量
            if len(all_hotels) > max_hotels:
                all_hotels = all_hotels[:max_hotels]
            
            stats["total_hotels"] = len(all_hotels)
            stats["cities_searched"] = 1
            
            # 计算统计信息
            end_time = time.time()
            stats["total_time"] = round(end_time - start_time, 2)
            
            if stats["platforms_searched"] > 0 and stats["total_hotels"] > 0:
                stats["success_rate"] = 100
            else:
                stats["success_rate"] = 0
            
            # 按价格排序
            all_hotels.sort(key=lambda x: x.get("lowest_price", 0))
            
            # 生成分析报告
            analysis = self._analyze_hotel_data(all_hotels)
            
            result = {
                "data": all_hotels,
                "stats": stats,
                "analysis": analysis,
                "summary": {
                    "total": len(all_hotels),
                    "city": city,
                    "check_in": check_in,
                    "check_out": check_out,
                    "platforms": platform,
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                    "status": "success",
                    "average_price": analysis.get("price_stats", {}).get("average", 0),
                    "min_price": analysis.get("price_stats", {}).get("min", 0),
                    "max_price": analysis.get("price_stats", {}).get("max", 0)
                }
            }
            
            self.logger.info(f"旅游网站酒店价格爬虫运行完成")
            self.logger.info(f"总计: {len(all_hotels)} 家酒店, 用时: {stats['total_time']}秒")
            
            return result
            
        except Exception as e:
            self.logger.error(f"旅游网站酒店价格爬虫运行失败: {e}")
            
            return {
                "data": [],
                "stats": stats,
                "analysis": {},
                "summary": {
                    "total": 0,
                    "city": city,
                    "check_in": check_in,
                    "check_out": check_out,
                    "platforms": platform,
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                    "status": "error",
                    "error_message": str(e)
                }
            }
        finally:
            # 关闭Selenium驱动
            if self.driver:
                try:
                    self.driver.quit()
                    self.logger.info("Selenium驱动已关闭")
                except:
                    pass
    
    def _analyze_hotel_data(self, hotels: List[Dict]) -> Dict[str, Any]:
        """分析酒店数据"""
        if not hotels:
            return {}
        
        analysis = {
            "price_stats": {},
            "rating_stats": {},
            "star_distribution": {},
            "platform_distribution": {},
            "recommendations": []
        }
        
        # 价格统计
        prices = [h.get("lowest_price", 0) for h in hotels if h.get("lowest_price", 0) > 0]
        if prices:
            analysis["price_stats"] = {
                "min": min(prices),
                "max": max(prices),
                "average": round(sum(prices) / len(prices), 2),
                "median": sorted(prices)[len(prices) // 2] if len(prices) % 2 == 1 else 
                         (sorted(prices)[len(prices) // 2 - 1] + sorted(prices)[len(prices) // 2]) / 2
            }
        
        # 评分统计
        ratings = [h.get("user_rating", 0) for h in hotels if h.get("user_rating", 0) > 0]
        if ratings:
            analysis["rating_stats"] = {
                "min": min(ratings),
                "max": max(ratings),
                "average": round(sum(ratings) / len(ratings), 2),
                "count": len(ratings)
            }
        
        # 星级分布
        star_counts = {}
        for hotel in hotels:
            star = hotel.get("star_rating", 0)
            star_counts[star] = star_counts.get(star, 0) + 1
        analysis["star_distribution"] = star_counts
        
        # 平台分布
        platform_counts = {}
        for hotel in hotels:
            platform = hotel.get("platform", "未知")
            platform_counts[platform] = platform_counts.get(platform, 0) + 1
        analysis["platform_distribution"] = platform_counts
        
        # 推荐酒店（综合评分和价格）
        if hotels:
            # 计算性价比得分 (评分/价格 * 1000)
            scored_hotels = []
            for hotel in hotels:
                rating = hotel.get("user_rating", 0)
                price = hotel.get("lowest_price", 1)
                if rating > 0 and price > 0:
                    score = (rating / price) * 1000
                else:
                    score = 0
                
                scored_hotels.append({
                    "hotel": hotel.get("hotel_name", "未知"),
                    "platform": hotel.get("platform", "未知"),
                    "rating": rating,
                    "price": price,
                    "score": round(score, 2),
                    "address": hotel.get("address", "")
                })
            
            # 按性价比排序
            scored_hotels.sort(key=lambda x: x["score"], reverse=True)
            analysis["recommendations"] = scored_hotels[:10]  # 前10个推荐
        
        return analysis
    
    def export_data(self, data: Dict[str, Any], filename: str = None) -> str:
        """
        导出酒店数据到Excel
        
        Args:
            data: 要导出的数据
            filename: 输出文件名
            
        Returns:
            导出的文件路径
        """
        try:
            import pandas as pd
            from pathlib import Path
            
            # 准备输出目录
            output_dir = Path("data") / "travel"
            output_dir.mkdir(parents=True, exist_ok=True)
            
            # 生成文件名
            if not filename:
                timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                city = data.get("summary", {}).get("city", "unknown")
                filename = f"hotel_prices_{city}_{timestamp}.xlsx"
            
            filepath = output_dir / filename
            
            # 准备酒店数据
            hotels_data = data.get("data", [])
            if not hotels_data:
                self.logger.warning("没有酒店数据可导出")
                return ""
            
            # 转换为DataFrame
            df_hotels = pd.DataFrame(hotels_data)
            
            # 准备分析数据
            analysis = data.get("analysis", {})
            
            # 创建Excel写入器
            with pd.ExcelWriter(filepath, engine='openpyxl') as writer:
                # 写入酒店数据
                df_hotels.to_excel(writer, sheet_name='酒店列表', index=False)
                
                # 写入价格统计
                price_stats = analysis.get("price_stats", {})
                if price_stats:
                    df_prices = pd.DataFrame([price_stats])
                    df_prices.to_excel(writer, sheet_name='价格统计', index=False)
                
                # 写入星级分布
                star_dist = analysis.get("star_distribution", {})
                if star_dist:
                    df_stars = pd.DataFrame(list(star_dist.items()), columns=['星级', '数量'])
                    df_stars.to_excel(writer, sheet_name='星级分布', index=False)
                
                # 写入平台分布
                platform_dist = analysis.get("platform_distribution", {})
                if platform_dist:
                    df_platforms = pd.DataFrame(list(platform_dist.items()), columns=['平台', '数量'])
                    df_platforms.to_excel(writer, sheet_name='平台分布', index=False)
                
                # 写入推荐酒店
                recommendations = analysis.get("recommendations", [])
                if recommendations:
                    df_rec = pd.DataFrame(recommendations)
                    df_rec.to_excel(writer, sheet_name='推荐酒店', index=False)
                
                # 写入汇总信息
                summary = data.get("summary", {})
                df_summary = pd.DataFrame([summary])
                df_summary.to_excel(writer, sheet_name='汇总信息', index=False)
            
            self.logger.info(f"酒店数据已导出到: {filepath}")
            return str(filepath)
            
        except Exception as e:
            self.logger.error(f"导出酒店数据失败: {e}")
            return ""


if __name__ == "__main__":
    # 演示用法
    crawler = TravelCrawler()
    
    # 运行爬虫
    data = crawler.run()
    
    # 导出数据
    if data:
        filepath = crawler.export_data(data)
        print(f"\n数据已导出到: {filepath}")
