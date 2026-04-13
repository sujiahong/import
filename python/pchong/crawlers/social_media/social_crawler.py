#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
社交媒体情感分析爬虫 (CR003)
爬取微博、知乎、小红书、抖音等社交媒体内容，并进行情感分析
"""

import re
import time
import json
import random
from datetime import datetime
from typing import Dict, List, Optional, Tuple, Any
import requests
from bs4 import BeautifulSoup
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, NoSuchElementException

from crawlers.base.base_crawler import BaseCrawler
from config import settings, specific_config


class SocialMediaCrawler(BaseCrawler):
    """社交媒体情感分析爬虫"""
    
    def __init__(self, output_dir: str = None):
        """
        初始化社交媒体爬虫
        
        Args:
            output_dir: 输出目录
        """
        super().__init__(output_dir)
        self.config = specific_config.SOCIAL_MEDIA
        self.driver = None
        self.sentiment_keywords = {
            "positive": ["好", "喜欢", "赞", "支持", "优秀", "棒", "开心", "幸福", "感谢", "推荐"],
            "negative": ["差", "讨厌", "垃圾", "失望", "糟糕", "问题", "投诉", "生气", "后悔", "不推荐"],
            "neutral": ["一般", "普通", "还行", "可以", "正常", "一般般", "普通", "中等"]
        }
    
    def _init_driver(self):
        """初始化Selenium WebDriver"""
        if self.driver is None:
            chrome_options = Options()
            
            # 添加反爬虫对抗选项
            chrome_options.add_argument("--disable-blink-features=AutomationControlled")
            chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
            chrome_options.add_experimental_option("useAutomationExtension", False)
            chrome_options.add_argument("--disable-gpu")
            chrome_options.add_argument("--no-sandbox")
            chrome_options.add_argument("--disable-dev-shm-usage")
            
            # 设置随机User-Agent
            user_agent = random.choice(settings.USER_AGENTS)
            chrome_options.add_argument(f"user-agent={user_agent}")
            
            # 可选：无头模式
            if self.config.get("headless", False):
                chrome_options.add_argument("--headless")
            
            self.driver = webdriver.Chrome(options=chrome_options)
            
            # 执行JavaScript隐藏自动化特征
            self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
    
    def _close_driver(self):
        """关闭WebDriver"""
        if self.driver:
            self.driver.quit()
            self.driver = None
    
    def analyze_sentiment(self, text: str) -> Dict[str, Any]:
        """
        分析文本情感
        
        Args:
            text: 待分析文本
            
        Returns:
            情感分析结果
        """
        if not text:
            return {"sentiment": "neutral", "score": 0.0, "keywords": []}
        
        text_lower = text.lower()
        positive_count = 0
        negative_count = 0
        found_keywords = []
        
        # 统计正面关键词
        for keyword in self.sentiment_keywords["positive"]:
            if keyword in text:
                positive_count += 1
                found_keywords.append(f"positive:{keyword}")
        
        # 统计负面关键词
        for keyword in self.sentiment_keywords["negative"]:
            if keyword in text:
                negative_count += 1
                found_keywords.append(f"negative:{keyword}")
        
        # 计算情感分数
        total_keywords = positive_count + negative_count
        if total_keywords == 0:
            sentiment = "neutral"
            score = 0.0
        else:
            score = (positive_count - negative_count) / total_keywords
            if score > 0.2:
                sentiment = "positive"
            elif score < -0.2:
                sentiment = "negative"
            else:
                sentiment = "neutral"
        
        return {
            "sentiment": sentiment,
            "score": round(score, 2),
            "keywords": found_keywords,
            "positive_count": positive_count,
            "negative_count": negative_count
        }
    
    def crawl_weibo(self, keyword: str = None, max_posts: int = 50) -> List[Dict]:
        """
        爬取微博数据
        
        Args:
            keyword: 搜索关键词
            max_posts: 最大帖子数
            
        Returns:
            微博数据列表
        """
        print(f"开始爬取微博数据，关键词: {keyword or '热门'}，最大数量: {max_posts}")
        
        try:
            self._init_driver()
            
            # 构建微博搜索URL
            if keyword:
                search_url = f"https://s.weibo.com/weibo?q={keyword}&Refer=index"
            else:
                search_url = "https://s.weibo.com/top/summary"
            
            self.driver.get(search_url)
            time.sleep(random.uniform(2, 4))
            
            # 模拟人类滚动
            for _ in range(3):
                self.driver.execute_script("window.scrollBy(0, 500);")
                time.sleep(random.uniform(1, 2))
            
            # 解析页面
            soup = BeautifulSoup(self.driver.page_source, 'html.parser')
            posts = []
            
            # 尝试不同的选择器
            selectors = [
                "div.card",  # 微博卡片
                "div.card-wrap",  # 微博卡片包装
                "article",  # 文章元素
                "div[action-type='feed_list_item']"  # 微博列表项
            ]
            
            for selector in selectors:
                elements = soup.select(selector)
                if elements:
                    break
            
            for i, element in enumerate(elements[:max_posts]):
                try:
                    post_data = self._parse_weibo_element(element)
                    if post_data:
                        posts.append(post_data)
                        print(f"  爬取微博帖子 {i+1}: {post_data.get('content', '')[:50]}...")
                except Exception as e:
                    print(f"  解析微博帖子 {i+1} 时出错: {e}")
                
                # 随机延迟
                time.sleep(random.uniform(0.5, 1.5))
            
            print(f"微博爬取完成，共获取 {len(posts)} 条数据")
            return posts
            
        except Exception as e:
            print(f"爬取微博时出错: {e}")
            return []
        finally:
            self._close_driver()
    
    def _parse_weibo_element(self, element) -> Optional[Dict]:
        """解析微博元素"""
        try:
            # 提取内容
            content_elem = element.select_one("p.txt")
            content = content_elem.get_text(strip=True) if content_elem else ""
            
            # 提取用户信息
            user_elem = element.select_one("a.name")
            username = user_elem.get_text(strip=True) if user_elem else ""
            user_url = user_elem.get("href", "") if user_elem else ""
            
            # 提取时间
            time_elem = element.select_one("a.date")
            post_time = time_elem.get_text(strip=True) if time_elem else ""
            
            # 提取互动数据
            like_elem = element.select_one("span[node-type='like_status']")
            like_count = like_elem.get_text(strip=True) if like_elem else "0"
            
            repost_elem = element.select_one("span[node-type='forward_btn_text']")
            repost_count = repost_elem.get_text(strip=True) if repost_elem else "0"
            
            comment_elem = element.select_one("span[node-type='comment_btn_text']")
            comment_count = comment_elem.get_text(strip=True) if comment_elem else "0"
            
            # 情感分析
            sentiment = self.analyze_sentiment(content)
            
            return {
                "platform": "weibo",
                "username": username,
                "content": content,
                "post_time": post_time,
                "like_count": self._parse_count(like_count),
                "repost_count": self._parse_count(repost_count),
                "comment_count": self._parse_count(comment_count),
                "sentiment": sentiment["sentiment"],
                "sentiment_score": sentiment["score"],
                "sentiment_keywords": sentiment["keywords"],
                "user_url": f"https:{user_url}" if user_url.startswith("//") else user_url,
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            }
        except Exception as e:
            print(f"解析微博元素时出错: {e}")
            return None
    
    def crawl_zhihu(self, topic: str = None, max_answers: int = 30) -> List[Dict]:
        """
        爬取知乎数据
        
        Args:
            topic: 话题/问题
            max_answers: 最大回答数
            
        Returns:
            知乎数据列表
        """
        print(f"开始爬取知乎数据，话题: {topic or '热门'}，最大数量: {max_answers}")
        
        try:
            self._init_driver()
            
            # 构建知乎URL
            if topic:
                search_url = f"https://www.zhihu.com/search?type=content&q={topic}"
            else:
                search_url = "https://www.zhihu.com/hot"
            
            self.driver.get(search_url)
            time.sleep(random.uniform(3, 5))
            
            # 模拟滚动
            for _ in range(5):
                self.driver.execute_script("window.scrollBy(0, 800);")
                time.sleep(random.uniform(1.5, 2.5))
            
            # 解析页面
            soup = BeautifulSoup(self.driver.page_source, 'html.parser')
            answers = []
            
            # 知乎回答选择器
            selectors = [
                "div.List-item",  # 回答列表项
                "div.AnswerItem",  # 回答项
                "div.ContentItem",  # 内容项
                "div[data-za-detail-view-path-module='AnswerItem']"  # 回答项
            ]
            
            for selector in selectors:
                elements = soup.select(selector)
                if elements:
                    break
            
            for i, element in enumerate(elements[:max_answers]):
                try:
                    answer_data = self._parse_zhihu_element(element)
                    if answer_data:
                        answers.append(answer_data)
                        print(f"  爬取知乎回答 {i+1}: {answer_data.get('content', '')[:50]}...")
                except Exception as e:
                    print(f"  解析知乎回答 {i+1} 时出错: {e}")
                
                time.sleep(random.uniform(0.5, 1.5))
            
            print(f"知乎爬取完成，共获取 {len(answers)} 条数据")
            return answers
            
        except Exception as e:
            print(f"爬取知乎时出错: {e}")
            return []
        finally:
            self._close_driver()
    
    def _parse_zhihu_element(self, element) -> Optional[Dict]:
        """解析知乎元素"""
        try:
            # 提取内容
            content_elem = element.select_one("div.RichContent-inner")
            content = content_elem.get_text(strip=True) if content_elem else ""
            
            # 提取问题标题
            question_elem = element.select_one("h2.ContentItem-title a")
            question = question_elem.get_text(strip=True) if question_elem else ""
            question_url = question_elem.get("href", "") if question_elem else ""
            
            # 提取作者信息
            author_elem = element.select_one("a.UserLink-link")
            author = author_elem.get_text(strip=True) if author_elem else ""
            author_url = author_elem.get("href", "") if author_elem else ""
            
            # 提取点赞数
            upvote_elem = element.select_one("button.VoteButton--up")
            upvote_text = upvote_elem.get_text(strip=True) if upvote_elem else "0"
            upvote_count = self._parse_zhihu_count(upvote_text)
            
            # 提取评论数
            comment_elem = element.select_one("button.ContentItem-action[aria-label^='评论']")
            comment_text = comment_elem.get_text(strip=True) if comment_elem else "0"
            comment_count = self._parse_zhihu_count(comment_text)
            
            # 情感分析
            sentiment = self.analyze_sentiment(content)
            
            return {
                "platform": "zhihu",
                "question": question,
                "content": content,
                "author": author,
                "upvote_count": upvote_count,
                "comment_count": comment_count,
                "sentiment": sentiment["sentiment"],
                "sentiment_score": sentiment["score"],
                "sentiment_keywords": sentiment["keywords"],
                "question_url": f"https://www.zhihu.com{question_url}" if question_url else "",
                "author_url": f"https://www.zhihu.com{author_url}" if author_url else "",
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            }
        except Exception as e:
            print(f"解析知乎元素时出错: {e}")
            return None
    
    def _parse_zhihu_count(self, text: str) -> int:
        """解析知乎计数文本"""
        try:
            if "K" in text or "k" in text:
                num = float(text.replace('K', '').replace('k', '')) * 1000
            elif "万" in text:
                num = float(text.replace('万', '')) * 10000
            else:
                num = float(text)
            return int(num)
        except:
            return 0
    
    def crawl_xiaohongshu(self, keyword: str = None, max_notes: int = 20) -> List[Dict]:
        """
        爬取小红书数据
        
        Args:
            keyword: 搜索关键词
            max_notes: 最大笔记数
            
        Returns:
            小红书数据列表
        """
        print(f"开始爬取小红书数据，关键词: {keyword or '推荐'}，最大数量: {max_notes}")
        
        try:
            # 小红书有较强的反爬虫，这里使用API模拟
            # 注意：实际使用时需要处理登录和反爬虫
            
            # 使用模拟数据演示
            notes = []
            for i in range(min(max_notes, 10)):
                note_data = self._generate_mock_xiaohongshu_data(i, keyword)
                notes.append(note_data)
                print(f"  生成小红书笔记 {i+1}: {note_data.get('title', '')[:30]}...")
            
            print(f"小红书爬取完成，共获取 {len(notes)} 条数据")
            return notes
            
        except Exception as e:
            print(f"爬取小红书时出错: {e}")
            return []
    
    def _generate_mock_xiaohongshu_data(self, index: int, keyword: str = None) -> Dict:
        """生成小红书模拟数据"""
        titles = [
            "周末出游穿搭分享",
            "美食探店必吃榜",
            "护肤心得大公开",
            "家居好物推荐",
            "旅行攻略收藏",
            "健身减肥计划",
            "职场穿搭技巧",
            "亲子活动推荐",
            "美妆教程分享",
            "读书笔记整理"
        ]
        
        contents = [
            "这家店真的太好吃了！强烈推荐他们的招牌菜，味道绝绝子！",
            "这次旅行真的超值，风景美到让人窒息，强烈推荐给大家！",
            "这个护肤方法我用了半年，皮肤真的变好了很多！",
            "最近发现的家居好物，让生活变得更加美好！",
            "这次购物体验一般，感觉没有想象中的那么好。",
            "产品有些小问题，客服态度也不是很好，有点失望。",
            "整体来说还可以，性价比还算不错，可以考虑购买。",
            "非常棒的产品，已经回购第三次了，强烈推荐！"
        ]
        
        authors = ["小红薯", "美妆达人", "旅行家", "美食家", "穿搭博主", "家居达人", "健身教练", "读书爱好者"]
        
        title = random.choice(titles)
        if keyword:
            title = f"{keyword}相关 - {title}"
        
        content = random.choice(contents)
        sentiment = self.analyze_sentiment(content)
        
        return {
            "platform": "xiaohongshu",
            "title": title,
            "content": content,
            "author": random.choice(authors),
            "like_count": random.randint(100, 10000),
            "collect_count": random.randint(50, 5000),
            "comment_count": random.randint(10, 1000),
            "tags": ["推荐", "分享", "好物"],
            "sentiment": sentiment["sentiment"],
            "sentiment_score": sentiment["score"],
            "sentiment_keywords": sentiment["keywords"],
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
    
    def crawl_douyin(self, hashtag: str = None, max_videos: int = 15) -> List[Dict]:
        """
        爬取抖音数据
        
        Args:
            hashtag: 话题标签
            max_videos: 最大视频数
            
        Returns:
            抖音数据列表
        """
        print(f"开始爬取抖音数据，话题: {hashtag or '热门'}，最大数量: {max_videos}")
        
        try:
            # 抖音有很强的反爬虫，这里使用API模拟
            # 实际使用时需要处理签名、加密等
            
            videos = []
            for i in range(min(max_videos, 10)):
                video_data = self._generate_mock_douyin_data(i, hashtag)
                videos.append(video_data)
                print(f"  生成抖音视频 {i+1}: {video_data.get('description', '')[:30]}...")
            
            print(f"抖音爬取完成，共获取 {len(videos)} 条数据")
            return videos
            
        except Exception as e:
            print(f"爬取抖音时出错: {e}")
            return []
    
    def _generate_mock_douyin_data(self, index: int, hashtag: str = None) -> Dict:
        """生成抖音模拟数据"""
        descriptions = [
            "这个舞蹈太绝了！ #舞蹈挑战",
            "美食制作教程，简单易学！ #美食教程",
            "旅行vlog分享，风景美如画！ #旅行",
            "搞笑日常，笑到肚子疼！ #搞笑",
            "宠物日常，太可爱了！ #宠物",
            "健身打卡第30天，继续努力！ #健身",
            "美妆教程，新手也能学会！ #美妆",
            "生活小技巧，太实用了！ #生活技巧"
        ]
        
        authors = ["抖音达人", "美食博主", "旅行家", "搞笑主播", "宠物博主", "健身教练", "美妆达人", "生活家"]
        
        hashtags = ["热门", "挑战", "教程", "分享", "日常", "打卡"]
        if hashtag:
            hashtags.append(hashtag)
        
        description = random.choice(descriptions)
        sentiment = self.analyze_sentiment(description)
        
        return {
            "platform": "douyin",
            "description": description,
            "author": random.choice(authors),
            "like_count": random.randint(1000, 1000000),
            "comment_count": random.randint(100, 100000),
            "share_count": random.randint(50, 50000),
            "hashtags": hashtags,
            "duration": f"{random.randint(15, 60)}秒",
            "sentiment": sentiment["sentiment"],
            "sentiment_score": sentiment["score"],
            "sentiment_keywords": sentiment["keywords"],
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
    
    def _parse_count(self, text: str) -> int:
        """解析计数文本（万/K转换）"""
        try:
            text = text.strip()
            if not text:
                return 0
            
            # 处理中文计数
            if "万" in text:
                num = float(text.replace('万', '')) * 10000
            elif "K" in text or "k" in text:
                num = float(text.replace('K', '').replace('k', '')) * 1000
            else:
                # 移除非数字字符
                num_text = re.sub(r'[^\d.]', '', text)
                num = float(num_text) if num_text else 0
            
            return int(num)
        except Exception as e:
            print(f"解析计数 '{text}' 时出错: {e}")
            return 0
    
    def run(self, 
            platforms: List[str] = None,
            keywords: Dict[str, str] = None,
            max_items: int = 100,
            enable_sentiment: bool = True,
            **kwargs) -> List[Dict]:
        """
        运行社交媒体爬虫
        
        Args:
            platforms: 平台列表，可选 ["weibo", "zhihu", "xiaohongshu", "douyin"]
            keywords: 各平台关键词，如 {"weibo": "科技", "zhihu": "人工智能"}
            max_items: 每个平台最大项目数
            enable_sentiment: 是否启用情感分析
            
        Returns:
            社交媒体数据列表
        """
        print("=" * 60)
        print("开始运行社交媒体情感分析爬虫")
        print("=" * 60)
        
        # 设置默认值
        if platforms is None:
            platforms = ["weibo", "zhihu"]
        
        if keywords is None:
            keywords = {}
        
        all_data = []
        start_time = datetime.now()
        
        try:
            # 爬取微博
            if "weibo" in platforms:
                print(f"\n📱 爬取微博...")
                weibo_keyword = keywords.get("weibo", "科技")
                weibo_data = self.crawl_weibo(weibo_keyword, max_items // len(platforms))
                all_data.extend(weibo_data)
            
            # 爬取知乎
            if "zhihu" in platforms:
                print(f"\n📚 爬取知乎...")
                zhihu_topic = keywords.get("zhihu", "人工智能")
                zhihu_data = self.crawl_zhihu(zhihu_topic, max_items // len(platforms))
                all_data.extend(zhihu_data)
            
            # 爬取小红书
            if "xiaohongshu" in platforms:
                print(f"\n📕 爬取小红书...")
                xhs_keyword = keywords.get("xiaohongshu", "推荐")
                xhs_data = self.crawl_xiaohongshu(xhs_keyword, max_items // len(platforms))
                all_data.extend(xhs_data)
            
            # 爬取抖音
            if "douyin" in platforms:
                print(f"\n🎵 爬取抖音...")
                douyin_hashtag = keywords.get("douyin", "热门")
                douyin_data = self.crawl_douyin(douyin_hashtag, max_items // len(platforms))
                all_data.extend(douyin_data)
            
            # 情感分析汇总
            if enable_sentiment and all_data:
                sentiment_summary = self._analyze_sentiment_summary(all_data)
                print(f"\n📊 情感分析汇总:")
                print(f"   正面: {sentiment_summary['positive']} 条")
                print(f"   负面: {sentiment_summary['negative']} 条")
                print(f"   中性: {sentiment_summary['neutral']} 条")
                print(f"   总体情感: {sentiment_summary['overall_sentiment']}")
            
            # 生成报告
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()
            
            print(f"\n✅ 社交媒体爬取完成!")
            print(f"   总计: {len(all_data)} 条数据")
            print(f"   耗时: {duration:.2f} 秒")
            print(f"   平台: {', '.join(platforms)}")
            
            return all_data
            
        except Exception as e:
            print(f"❌ 社交媒体爬取失败: {e}")
            import traceback
            traceback.print_exc()
            return []
    
    def _analyze_sentiment_summary(self, data: List[Dict]) -> Dict:
        """分析情感汇总"""
        sentiment_counts = {"positive": 0, "negative": 0, "neutral": 0}
        
        for item in data:
            sentiment = item.get("sentiment", "neutral")
            if sentiment in sentiment_counts:
                sentiment_counts[sentiment] += 1
        
        total = sum(sentiment_counts.values())
        if total > 0:
            positive_ratio = sentiment_counts["positive"] / total
            negative_ratio = sentiment_counts["negative"] / total
            
            if positive_ratio > 0.6:
                overall = "strongly_positive"
            elif positive_ratio > 0.4:
                overall = "positive"
            elif negative_ratio > 0.6:
                overall = "strongly_negative"
            elif negative_ratio > 0.4:
                overall = "negative"
            else:
                overall = "neutral"
        else:
            overall = "neutral"
        
        return {
            "positive": sentiment_counts["positive"],
            "negative": sentiment_counts["negative"],
            "neutral": sentiment_counts["neutral"],
            "total": total,
            "overall_sentiment": overall
        }
    
    def export_data(self, data: List[Dict], filename: str = None) -> str:
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
            filename = f"social_media_data_{timestamp}.xlsx"
        
        # 准备数据
        export_data = []
        for item in data:
            export_item = {
                "平台": item.get("platform", ""),
                "作者/用户": item.get("username") or item.get("author") or item.get("用户", ""),
                "内容": item.get("content") or item.get("title") or item.get("description", ""),
                "发布时间": item.get("post_time", ""),
                "点赞数": item.get("like_count", 0),
                "评论数": item.get("comment_count", 0),
                "转发数": item.get("repost_count", 0),
                "情感倾向": item.get("sentiment", ""),
                "情感分数": item.get("sentiment_score", 0),
                "情感关键词": ", ".join(item.get("sentiment_keywords", [])),
                "链接": item.get("user_url") or item.get("question_url") or item.get("链接", ""),
                "爬取时间": item.get("crawl_time", "")
            }
            export_data.append(export_item)
        
        # 导出到Excel
        filepath = self.exporter.export_to_excel(
            data=export_data,
            filename=filename,
            sheet_name="社交媒体数据",
            title="社交媒体情感分析数据报告"
        )
        
        return filepath


if __name__ == "__main__":
    # 演示用法
    crawler = SocialMediaCrawler()
    
    # 运行爬虫
    data = crawler.run(
        platforms=["weibo", "zhihu"],
        keywords={"weibo": "科技", "zhihu": "人工智能"},
        max_items=20
    )
    
    # 导出数据
    if data:
        filepath = crawler.export_data(data)
        print(f"\n数据已导出到: {filepath}")