#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
股票金融数据爬虫 (CR006)
爬取东方财富、新浪财经、同花顺等金融数据
"""

import time
import json
import random
import re
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any
import requests
from bs4 import BeautifulSoup
import pandas as pd

from crawlers.base.base_crawler import BaseCrawler
from config import settings, specific_config


class FinanceCrawler(BaseCrawler):
    """股票金融数据爬虫"""
    
    def __init__(self, output_dir: str = None):
        """
        初始化金融数据爬虫
        
        Args:
            output_dir: 输出目录
        """
        super().__init__(output_dir)
        self.config = specific_config.FINANCE
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": random.choice(settings.USER_AGENTS),
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
            "Accept-Encoding": "gzip, deflate, br",
            "Connection": "keep-alive",
        })
    
    def get_stock_list(self, exchange: str = "sh") -> List[Dict]:
        """
        获取股票列表
        
        Args:
            exchange: 交易所，sh(上海), sz(深圳), bj(北京)
            
        Returns:
            股票列表
        """
        print(f"获取{exchange.upper()}交易所股票列表...")
        
        try:
            # 东方财富股票列表API
            if exchange == "sh":
                url = "http://quote.eastmoney.com/center/gridlist.html#sh_a_board"
            elif exchange == "sz":
                url = "http://quote.eastmoney.com/center/gridlist.html#sz_a_board"
            elif exchange == "bj":
                url = "http://quote.eastmoney.com/center/gridlist.html#bj_a_board"
            else:
                url = "http://quote.eastmoney.com/center/gridlist.html#hs_a_board"
            
            response = self._make_request(url)
            if not response:
                return []
            
            soup = BeautifulSoup(response.text, 'html.parser')
            stocks = []
            
            # 解析股票表格
            table = soup.find('table', {'id': 'table_wrapper-table'})
            if not table:
                # 尝试其他选择器
                table = soup.find('table', {'class': 'table'})
            
            if table:
                rows = table.find_all('tr')[1:]  # 跳过表头
                for row in rows:
                    try:
                        cols = row.find_all('td')
                        if len(cols) >= 6:
                            stock_data = {
                                "code": cols[1].text.strip(),
                                "name": cols[2].text.strip(),
                                "latest_price": self._parse_price(cols[3].text.strip()),
                                "change": self._parse_price(cols[4].text.strip()),
                                "change_percent": self._parse_percent(cols[5].text.strip()),
                                "volume": self._parse_volume(cols[6].text.strip()) if len(cols) > 6 else 0,
                                "turnover": self._parse_volume(cols[7].text.strip()) if len(cols) > 7 else 0,
                                "exchange": exchange.upper(),
                                "update_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                            }
                            stocks.append(stock_data)
                    except Exception as e:
                        print(f"解析股票行时出错: {e}")
            
            print(f"获取到 {len(stocks)} 只股票")
            return stocks[:100]  # 限制数量
            
        except Exception as e:
            print(f"获取股票列表时出错: {e}")
            return []
    
    def get_stock_real_time(self, stock_code: str) -> Optional[Dict]:
        """
        获取股票实时数据
        
        Args:
            stock_code: 股票代码，如 000001, 600000
            
        Returns:
            股票实时数据
        """
        print(f"获取股票 {stock_code} 实时数据...")
        
        try:
            # 新浪财经API
            if stock_code.startswith('6'):
                symbol = f"sh{stock_code}"
            elif stock_code.startswith('0') or stock_code.startswith('3'):
                symbol = f"sz{stock_code}"
            else:
                symbol = stock_code
            
            url = f"https://hq.sinajs.cn/list={symbol}"
            
            response = self._make_request(url)
            if not response:
                return None
            
            # 解析新浪财经数据格式
            content = response.text
            if not content or "=" not in content:
                return None
            
            data_str = content.split('="')[1].split('"')[0]
            data_fields = data_str.split(',')
            
            if len(data_fields) < 30:
                return None
            
            stock_data = {
                "code": stock_code,
                "name": data_fields[0],
                "open": self._parse_price(data_fields[1]),
                "pre_close": self._parse_price(data_fields[2]),
                "latest_price": self._parse_price(data_fields[3]),
                "high": self._parse_price(data_fields[4]),
                "low": self._parse_price(data_fields[5]),
                "bid_price": self._parse_price(data_fields[6]),
                "ask_price": self._parse_price(data_fields[7]),
                "volume": self._parse_volume(data_fields[8]),
                "turnover": self._parse_volume(data_fields[9]),
                "bid1_volume": self._parse_volume(data_fields[10]),
                "bid1_price": self._parse_price(data_fields[11]),
                "bid2_volume": self._parse_volume(data_fields[12]),
                "bid2_price": self._parse_price(data_fields[13]),
                "bid3_volume": self._parse_volume(data_fields[14]),
                "bid3_price": self._parse_price(data_fields[15]),
                "bid4_volume": self._parse_volume(data_fields[16]),
                "bid4_price": self._parse_price(data_fields[17]),
                "bid5_volume": self._parse_volume(data_fields[18]),
                "bid5_price": self._parse_price(data_fields[19]),
                "ask1_volume": self._parse_volume(data_fields[20]),
                "ask1_price": self._parse_price(data_fields[21]),
                "ask2_volume": self._parse_volume(data_fields[22]),
                "ask2_price": self._parse_price(data_fields[23]),
                "ask3_volume": self._parse_volume(data_fields[24]),
                "ask3_price": self._parse_price(data_fields[25]),
                "ask4_volume": self._parse_volume(data_fields[26]),
                "ask4_price": self._parse_price(data_fields[27]),
                "ask5_volume": self._parse_volume(data_fields[28]),
                "ask5_price": self._parse_price(data_fields[29]),
                "date": data_fields[30] if len(data_fields) > 30 else "",
                "time": data_fields[31] if len(data_fields) > 31 else "",
                "change": None,
                "change_percent": None
            }
            
            # 计算涨跌幅
            if stock_data["pre_close"] and stock_data["latest_price"]:
                stock_data["change"] = round(stock_data["latest_price"] - stock_data["pre_close"], 2)
                if stock_data["pre_close"] != 0:
                    stock_data["change_percent"] = round((stock_data["change"] / stock_data["pre_close"]) * 100, 2)
            
            stock_data["crawl_time"] = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            return stock_data
            
        except Exception as e:
            print(f"获取股票 {stock_code} 实时数据时出错: {e}")
            return None
    
    def get_stock_history(self, stock_code: str, days: int = 30) -> List[Dict]:
        """
        获取股票历史数据
        
        Args:
            stock_code: 股票代码
            days: 历史天数
            
        Returns:
            股票历史数据
        """
        print(f"获取股票 {stock_code} 最近{days}天历史数据...")
        
        try:
            # 东方财富历史数据API
            if stock_code.startswith('6'):
                symbol = f"SH{stock_code}"
            elif stock_code.startswith('0') or stock_code.startswith('3'):
                symbol = f"SZ{stock_code}"
            else:
                symbol = stock_code
            
            end_date = datetime.now().strftime("%Y%m%d")
            start_date = (datetime.now() - timedelta(days=days)).strftime("%Y%m%d")
            
            url = f"http://quotes.money.163.com/service/chddata.html"
            params = {
                "code": symbol,
                "start": start_date,
                "end": end_date,
                "fields": "TCLOSE;HIGH;LOW;TOPEN;LCLOSE;CHG;PCHG;TURNOVER;VOTURNOVER;VATURNOVER;TCAP;MCAP"
            }
            
            response = self._make_request(url, params=params)
            if not response or not response.text.strip():
                return []
            
            # 解析CSV数据
            lines = response.text.strip().split('\n')
            if len(lines) < 2:
                return []
            
            history_data = []
            headers = lines[0].split(',')
            
            for line in lines[1:]:
                try:
                    values = line.split(',')
                    if len(values) < 12:
                        continue
                    
                    history_item = {
                        "date": values[0].strip('"'),
                        "code": values[1].strip('"'),
                        "name": values[2].strip('"'),
                        "close": self._parse_price(values[3]),
                        "high": self._parse_price(values[4]),
                        "low": self._parse_price(values[5]),
                        "open": self._parse_price(values[6]),
                        "pre_close": self._parse_price(values[7]),
                        "change": self._parse_price(values[8]),
                        "change_percent": self._parse_percent(values[9]),
                        "turnover_rate": self._parse_percent(values[10]),
                        "volume": self._parse_volume(values[11]),
                        "amount": self._parse_volume(values[12]) if len(values) > 12 else 0,
                        "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    }
                    history_data.append(history_item)
                except Exception as e:
                    print(f"解析历史数据行时出错: {e}")
            
            print(f"获取到 {len(history_data)} 条历史数据")
            return history_data
            
        except Exception as e:
            print(f"获取股票 {stock_code} 历史数据时出错: {e}")
            return []
    
    def get_financial_news(self, max_news: int = 20) -> List[Dict]:
        """
        获取财经新闻
        
        Args:
            max_news: 最大新闻数
            
        Returns:
            财经新闻列表
        """
        print(f"获取财经新闻，最大数量: {max_news}")
        
        try:
            # 新浪财经新闻
            url = "https://finance.sina.com.cn"
            
            response = self._make_request(url)
            if not response:
                return []
            
            soup = BeautifulSoup(response.text, 'html.parser')
            news_list = []
            
            # 查找新闻链接
            news_selectors = [
                "div.blk_02",  # 新浪财经新闻块
                "div.blk_12",  # 财经新闻
                "ul.list_009",  # 新闻列表
                "div.news-item",  # 新闻项
                "a[href*='/roll/']",  # 滚动新闻
                "a[href*='finance.sina.com.cn']"  # 财经链接
            ]
            
            news_links = []
            for selector in news_selectors:
                elements = soup.select(selector)
                for elem in elements:
                    if elem.name == 'a' and elem.get('href'):
                        link = elem.get('href')
                        if link.startswith('http') and 'finance.sina.com.cn' in link:
                            news_links.append((link, elem.get_text(strip=True)))
                    else:
                        # 查找子链接
                        sub_links = elem.find_all('a', href=True)
                        for sub_link in sub_links:
                            link = sub_link.get('href')
                            if link.startswith('http') and 'finance.sina.com.cn' in link:
                                news_links.append((link, sub_link.get_text(strip=True)))
            
            # 去重
            unique_links = []
            seen = set()
            for link, title in news_links:
                if link not in seen and title:
                    seen.add(link)
                    unique_links.append((link, title))
            
            # 获取新闻详情
            for i, (link, title) in enumerate(unique_links[:max_news]):
                try:
                    print(f"  获取新闻 {i+1}: {title[:50]}...")
                    
                    news_response = self._make_request(link)
                    if not news_response:
                        continue
                    
                    news_soup = BeautifulSoup(news_response.text, 'html.parser')
                    
                    # 提取内容
                    content_elem = news_soup.select_one("div#artibody") or news_soup.select_one("div.article")
                    content = content_elem.get_text(strip=True) if content_elem else ""
                    
                    # 提取时间
                    time_elem = news_soup.select_one("span.date") or news_soup.select_one("div.artInfo")
                    publish_time = time_elem.get_text(strip=True) if time_elem else ""
                    
                    # 提取来源
                    source_elem = news_soup.select_one("span.source") or news_soup.select_one("div.source")
                    source = source_elem.get_text(strip=True) if source_elem else ""
                    
                    news_data = {
                        "title": title,
                        "url": link,
                        "content": content[:500] + "..." if len(content) > 500 else content,
                        "publish_time": publish_time,
                        "source": source,
                        "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    }
                    news_list.append(news_data)
                    
                    time.sleep(random.uniform(0.5, 1.5))
                    
                except Exception as e:
                    print(f"  获取新闻详情时出错: {e}")
            
            print(f"获取到 {len(news_list)} 条财经新闻")
            return news_list
            
        except Exception as e:
            print(f"获取财经新闻时出错: {e}")
            return []
    
    def get_market_indices(self) -> Dict[str, Any]:
        """
        获取市场指数
        
        Returns:
            市场指数数据
        """
        print("获取市场指数数据...")
        
        try:
            indices = {}
            
            # 上证指数
            sh_data = self.get_stock_real_time("000001")
            if sh_data:
                indices["shanghai"] = {
                    "name": "上证指数",
                    "value": sh_data.get("latest_price", 0),
                    "change": sh_data.get("change", 0),
                    "change_percent": sh_data.get("change_percent", 0),
                    "update_time": sh_data.get("time", "")
                }
            
            # 深证成指
            sz_data = self.get_stock_real_time("399001")
            if sz_data:
                indices["shenzhen"] = {
                    "name": "深证成指",
                    "value": sz_data.get("latest_price", 0),
                    "change": sz_data.get("change", 0),
                    "change_percent": sz_data.get("change_percent", 0),
                    "update_time": sz_data.get("time", "")
                }
            
            # 创业板指
                cyb_data = self.get_stock_real_time("399006")
            if cyb_data:
                indices["chuangyeban"] = {
                    "name": "创业板指",
                    "value": cyb_data.get("latest_price", 0),
                    "change": cyb_data.get("change", 0),
                    "change_percent": cyb_data.get("change_percent", 0),
                    "update_time": cyb_data.get("time", "")
                }
            
            # 计算市场情绪
            if indices:
                total_change = sum(idx.get("change_percent", 0) for idx in indices.values())
                avg_change = total_change / len(indices)
                
                if avg_change > 1:
                    market_sentiment = "bullish"
                elif avg_change < -1:
                    market_sentiment = "bearish"
                elif avg_change > 0.2:
                    market_sentiment = "slightly_bullish"
                elif avg_change < -0.2:
                    market_sentiment = "slightly_bearish"
                else:
                    market_sentiment = "neutral"
                
                indices["market_sentiment"] = market_sentiment
                indices["average_change"] = round(avg_change, 2)
            
            indices["crawl_time"] = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            print(f"获取到 {len(indices) - 2} 个市场指数")  # 减去sentiment和crawl_time
            
            return indices
            
        except Exception as e:
            print(f"获取市场指数时出错: {e}")
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
                
                response = self.session.get(url, params=params, timeout=10)
                
                if response.status_code == 200:
                    return response
                else:
                    print(f"请求失败，状态码: {response.status_code}, 尝试 {attempt+1}/{max_retries}")
                    
            except requests.exceptions.RequestException as e:
                print(f"请求异常: {e}, 尝试 {attempt+1}/{max_retries}")
            
            except Exception as e:
                print(f"未知错误: {e}, 尝试 {attempt+1}/{max_retries}")
        
        return None
    
    def _parse_price(self, text: str) -> Optional[float]:
        """解析价格文本"""
        try:
            if not text or text.strip() == "" or text.strip() == "-":
                return None
            text = text.replace(',', '').replace('¥', '').replace('$', '')
            return float(text.strip())
        except:
            return None
    
    def _parse_percent(self, text: str) -> Optional[float]:
        """解析百分比文本"""
        try:
            if not text or text.strip() == "" or text.strip() == "-":
                return None
            text = text.replace('%', '').replace('％', '')
            return float(text.strip())
        except:
            return None
    
    def _parse_volume(self, text: str) -> Optional[float]:
        """解析成交量/成交额文本"""
        try:
            if not text or text.strip() == "" or text.strip() == "-":
                return None
            
            text = text.replace(',', '').strip().lower()
            
            # 处理中文单位
            if '亿' in text:
                num = float(text.replace('亿', '')) * 100000000
            elif '万' in text:
                num = float(text.replace('万', '')) * 10000
            elif 'k' in text or 'k' in text:
                num = float(text.replace('k', '').replace('K', '')) * 1000
            elif 'm' in text:
                num = float(text.replace('m', '').replace('M', '')) * 1000000
            else:
                num = float(text)
            
            return num
        except:
            return None
    
    def run(self,
            data_types: List[str] = None,
            stock_codes: List[str] = None,
            days_history: int = 30,
            max_news: int = 10,
            **kwargs) -> Dict[str, Any]:
        """
        运行金融数据爬虫
        
        Args:
            data_types: 数据类型，可选 ["stock_list", "realtime", "history", "news", "indices"]
            stock_codes: 股票代码列表
            days_history: 历史数据天数
            max_news: 最大新闻数
            
        Returns:
            金融数据字典
        """
        print("=" * 60)
        print("开始运行股票金融数据爬虫")
        print("=" * 60)
        
        # 设置默认值
        if data_types is None:
            data_types = ["stock_list", "realtime", "indices"]
        
        if stock_codes is None:
            stock_codes = ["000001", "600000", "000858", "600519"]  # 平安银行、浦发银行、五粮液、贵州茅台
        
        all_data = {}
        start_time = datetime.now()
        
        try:
            # 获取股票列表
            if "stock_list" in data_types:
                print(f"\n📈 获取股票列表...")
                sh_stocks = self.get_stock_list("sh")
                sz_stocks = self.get_stock_list("sz")
                all_data["stock_list"] = sh_stocks + sz_stocks
                print(f"   获取到 {len(all_data['stock_list'])} 只股票")
            
            # 获取实时数据
            if "realtime" in data_types:
                print(f"\n⏰ 获取股票实时数据...")
                realtime_data = []
                for code in stock_codes:
                    stock_data = self.get_stock_real_time(code)
                    if stock_data:
                        realtime_data.append(stock_data)
                        print(f"   获取 {code}: {stock_data.get('name', '')} - {stock_data.get('latest_price', 0)}")
                    time.sleep(random.uniform(0.5, 1))
                all_data["realtime"] = realtime_data
            
            # 获取历史数据
            if "history" in data_types and stock_codes:
                print(f"\n📅 获取股票历史数据...")
                history_data = []
                for code in stock_codes[:3]:  # 限制数量
                    hist_data = self.get_stock_history(code, days_history)
                    history_data.extend(hist_data)
                    print(f"   获取 {code} 最近{days_history}天历史数据: {len(hist_data)} 条")
                    time.sleep(random.uniform(1, 2))
                all_data["history"] = history_data
            
            # 获取财经新闻
            if "news" in data_types:
                print(f"\n📰 获取财经新闻...")
                news_data = self.get_financial_news(max_news)
                all_data["news"] = news_data
                print(f"   获取到 {len(news_data)} 条财经新闻")
            
            # 获取市场指数
            if "indices" in data_types:
                print(f"\n📊 获取市场指数...")
                indices_data = self.get_market_indices()
                all_data["indices"] = indices_data
            
            # 生成报告
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()
            
            print(f"\n✅ 金融数据爬取完成!")
            print(f"   耗时: {duration:.2f} 秒")
            print(f"   数据类型: {', '.join(data_types)}")
            print(f"   股票数量: {len(stock_codes)}")
            
            # 统计信息
            stats = {}
            for key, value in all_data.items():
                if isinstance(value, list):
                    stats[key] = len(value)
                elif isinstance(value, dict):
                    stats[key] = f"{len(value)} 项"
            
            print(f"   数据统计: {stats}")
            
            return all_data
            
        except Exception as e:
            print(f"❌ 金融数据爬取失败: {e}")
            import traceback
            traceback.print_exc()
            return {}
    
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
            filename = f"finance_data_{timestamp}.xlsx"
        
        with pd.ExcelWriter(self.output_dir / filename, engine='openpyxl') as writer:
            # 导出股票列表
            if "stock_list" in data and data["stock_list"]:
                stock_df = pd.DataFrame(data["stock_list"])
                stock_df.to_excel(writer, sheet_name="股票列表", index=False)
                print(f"导出股票列表: {len(stock_df)} 行")
            
            # 导出实时数据
            if "realtime" in data and data["realtime"]:
                realtime_df = pd.DataFrame(data["realtime"])
                realtime_df.to_excel(writer, sheet_name="实时数据", index=False)
                print(f"导出实时数据: {len(realtime_df)} 行")
            
            # 导出历史数据
            if "history" in data and data["history"]:
                history_df = pd.DataFrame(data["history"])
                history_df.to_excel(writer, sheet_name="历史数据", index=False)
                print(f"导出历史数据: {len(history_df)} 行")
            
            # 导出财经新闻
            if "news" in data and data["news"]:
                news_df = pd.DataFrame(data["news"])
                news_df.to_excel(writer, sheet_name="财经新闻", index=False)
                print(f"导出财经新闻: {len(news_df)} 行")
            
            # 导出市场指数
            if "indices" in data and data["indices"]:
                # 转换字典为适合Excel的格式
                indices_list = []
                for key, value in data["indices"].items():
                    if isinstance(value, dict):
                        indices_list.append({
                            "指数名称": value.get("name", key),
                            "当前值": value.get("value", ""),
                            "涨跌": value.get("change", ""),
                            "涨跌幅%": value.get("change_percent", ""),
                            "更新时间": value.get("update_time", "")
                        })
                
                if indices_list:
                    indices_df = pd.DataFrame(indices_list)
                    indices_df.to_excel(writer, sheet_name="市场指数", index=False)
                    print(f"导出市场指数: {len(indices_df)} 行")
            
            # 添加汇总信息
            summary_data = {
                "数据项": ["爬取时间", "数据量", "导出文件"],
                "值": [
                    datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                    f"{sum(len(v) if isinstance(v, list) else 1 for v in data.values())} 项",
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
    crawler = FinanceCrawler()
    
    # 运行爬虫
    data = crawler.run(
        data_types=["stock_list", "realtime", "news", "indices"],
        stock_codes=["000001", "600000"],
        max_news=5
    )
    
    # 导出数据
    if data:
        filepath = crawler.export_data(data)
        print(f"\n数据已导出到: {filepath}")