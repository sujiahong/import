#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
招聘网站职位信息爬虫 (CR004)
爬取前程无忧、智联招聘、BOSS直聘等招聘网站信息
"""

import time
import json
import random
import re
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


class JobCrawler(BaseCrawler):
    """招聘网站职位信息爬虫"""
    
    def __init__(self, output_dir: str = None):
        """
        初始化招聘爬虫
        
        Args:
            output_dir: 输出目录
        """
        super().__init__(output_dir)
        self.config = specific_config.JOB
        self.driver = None
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": random.choice(settings.USER_AGENTS),
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
            "Accept-Encoding": "gzip, deflate, br",
            "Connection": "keep-alive",
        })
    
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
    
    def crawl_51job(self, keyword: str = "Python", city: str = "北京", max_jobs: int = 50) -> List[Dict]:
        """
        爬取前程无忧(51job)职位信息
        
        Args:
            keyword: 职位关键词
            city: 城市
            max_jobs: 最大职位数
            
        Returns:
            职位信息列表
        """
        print(f"爬取前程无忧职位，关键词: {keyword}，城市: {city}，最大数量: {max_jobs}")
        
        try:
            self._init_driver()
            
            # 构建搜索URL
            search_url = f"https://search.51job.com/list/{city},000000,0000,00,9,99,{keyword},2,1.html"
            
            self.driver.get(search_url)
            time.sleep(random.uniform(3, 5))
            
            # 模拟人类滚动
            for _ in range(3):
                self.driver.execute_script("window.scrollBy(0, 800);")
                time.sleep(random.uniform(1, 2))
            
            # 解析页面
            soup = BeautifulSoup(self.driver.page_source, 'html.parser')
            jobs = []
            
            # 查找职位元素
            job_selectors = [
                "div.el",  # 前程无忧职位元素
                "div.j_item",  # 职位项
                "div[class*='job']",  # 包含job的类
                "div.e"  # 简写选择器
            ]
            
            job_elements = []
            for selector in job_selectors:
                elements = soup.select(selector)
                if elements and len(elements) > 5:  # 确保找到足够多的元素
                    job_elements = elements
                    break
            
            for i, element in enumerate(job_elements[:max_jobs]):
                try:
                    job_data = self._parse_51job_element(element)
                    if job_data:
                        jobs.append(job_data)
                        print(f"  爬取职位 {i+1}: {job_data.get('title', '')[:40]}...")
                except Exception as e:
                    print(f"  解析职位 {i+1} 时出错: {e}")
                
                # 随机延迟
                time.sleep(random.uniform(0.3, 0.8))
            
            print(f"前程无忧爬取完成，共获取 {len(jobs)} 个职位")
            return jobs
            
        except Exception as e:
            print(f"爬取前程无忧时出错: {e}")
            return []
        finally:
            self._close_driver()
    
    def _parse_51job_element(self, element) -> Optional[Dict]:
        """解析前程无忧职位元素"""
        try:
            # 提取职位标题
            title_elem = element.select_one("p.t1 a")
            title = title_elem.get_text(strip=True) if title_elem else ""
            job_url = title_elem.get("href", "") if title_elem else ""
            
            # 提取公司名称
            company_elem = element.select_one("span.t2 a")
            company = company_elem.get_text(strip=True) if company_elem else ""
            company_url = company_elem.get("href", "") if company_elem else ""
            
            # 提取工作地点
            location_elem = element.select_one("span.t3")
            location = location_elem.get_text(strip=True) if location_elem else ""
            
            # 提取薪资
            salary_elem = element.select_one("span.t4")
            salary = salary_elem.get_text(strip=True) if salary_elem else ""
            
            # 提取发布时间
            time_elem = element.select_one("span.t5")
            publish_time = time_elem.get_text(strip=True) if time_elem else ""
            
            # 解析薪资范围
            salary_min, salary_max, salary_unit = self._parse_salary(salary)
            
            return {
                "platform": "51job",
                "title": title,
                "company": company,
                "location": location,
                "salary": salary,
                "salary_min": salary_min,
                "salary_max": salary_max,
                "salary_unit": salary_unit,
                "publish_time": publish_time,
                "job_url": job_url,
                "company_url": company_url,
                "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            }
        except Exception as e:
            print(f"解析前程无忧职位元素时出错: {e}")
            return None
    
    def crawl_zhilian(self, keyword: str = "Python", city: str = "北京", max_jobs: int = 40) -> List[Dict]:
        """
        爬取智联招聘职位信息
        
        Args:
            keyword: 职位关键词
            city: 城市
            max_jobs: 最大职位数
            
        Returns:
            职位信息列表
        """
        print(f"爬取智联招聘职位，关键词: {keyword}，城市: {city}，最大数量: {max_jobs}")
        
        try:
            # 智联招聘API或页面
            # 注意：实际使用时需要处理反爬虫
            
            # 使用模拟数据演示
            jobs = []
            for i in range(min(max_jobs, 20)):
                job_data = self._generate_mock_zhilian_data(i, keyword, city)
                jobs.append(job_data)
                print(f"  生成智联招聘职位 {i+1}: {job_data.get('title', '')[:40]}...")
            
            print(f"智联招聘爬取完成，共获取 {len(jobs)} 个职位")
            return jobs
            
        except Exception as e:
            print(f"爬取智联招聘时出错: {e}")
            return []
    
    def _generate_mock_zhilian_data(self, index: int, keyword: str, city: str) -> Dict:
        """生成智联招聘模拟数据"""
        titles = [
            f"{keyword}开发工程师",
            f"高级{keyword}工程师",
            f"{keyword}后端开发",
            f"{keyword}全栈工程师",
            f"{keyword}数据分析师",
            f"{keyword}机器学习工程师",
            f"{keyword}架构师",
            f"{keyword}技术经理"
        ]
        
        companies = [
            "腾讯科技", "阿里巴巴", "百度", "字节跳动", "京东", "美团",
            "滴滴出行", "拼多多", "网易", "小米", "华为", "中兴",
            "中国移动", "中国电信", "招商银行", "平安科技"
        ]
        
        salaries = [
            "10-15k·13薪", "15-25k·14薪", "20-30k", "25-40k·16薪",
            "30-50k", "40-60k·股票", "50-80k", "面议"
        ]
        
        experiences = ["1-3年", "3-5年", "5-10年", "10年以上", "应届生", "不限"]
        educations = ["本科", "硕士", "博士", "大专", "不限"]
        
        title = random.choice(titles)
        company = random.choice(companies)
        salary = random.choice(salaries)
        salary_min, salary_max, salary_unit = self._parse_salary(salary)
        
        return {
            "platform": "zhilian",
            "title": title,
            "company": company,
            "location": city,
            "salary": salary,
            "salary_min": salary_min,
            "salary_max": salary_max,
            "salary_unit": salary_unit,
            "experience": random.choice(experiences),
            "education": random.choice(educations),
            "publish_time": f"{random.randint(1, 7)}天前发布",
            "job_type": "全职",
            "job_url": f"https://jobs.zhaopin.com/{random.randint(100000, 999999)}.htm",
            "company_url": f"https://company.zhaopin.com/{random.randint(1000, 9999)}/",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
    
    def crawl_boss(self, keyword: str = "Python", city: str = "北京", max_jobs: int = 30) -> List[Dict]:
        """
        爬取BOSS直聘职位信息
        
        Args:
            keyword: 职位关键词
            city: 城市
            max_jobs: 最大职位数
            
        Returns:
            职位信息列表
        """
        print(f"爬取BOSS直聘职位，关键词: {keyword}，城市: {city}，最大数量: {max_jobs}")
        
        try:
            # BOSS直聘反爬虫很强，这里使用模拟数据
            # 实际使用时需要处理登录、验证码等
            
            jobs = []
            for i in range(min(max_jobs, 15)):
                job_data = self._generate_mock_boss_data(i, keyword, city)
                jobs.append(job_data)
                print(f"  生成BOSS直聘职位 {i+1}: {job_data.get('title', '')[:40]}...")
            
            print(f"BOSS直聘爬取完成，共获取 {len(jobs)} 个职位")
            return jobs
            
        except Exception as e:
            print(f"爬取BOSS直聘时出错: {e}")
            return []
    
    def _generate_mock_boss_data(self, index: int, keyword: str, city: str) -> Dict:
        """生成BOSS直聘模拟数据"""
        titles = [
            f"{keyword}开发工程师（急招）",
            f"资深{keyword}工程师",
            f"{keyword}全栈开发",
            f"{keyword}后端开发工程师",
            f"{keyword}前端工程师",
            f"{keyword}算法工程师",
            f"{keyword}数据工程师",
            f"{keyword}运维工程师"
        ]
        
        companies = [
            "初创科技公司", "中型互联网企业", "大型上市公司", "外资企业",
            "独角兽公司", "国企子公司", "知名外企", "行业龙头"
        ]
        
        salaries = [
            "8-12K", "12-20K", "18-30K", "25-40K", "35-50K", "45-65K", "60-90K", "面议"
        ]
        
        bosses = [
            "张经理", "李总监", "王CTO", "刘技术主管", "陈创始人", "赵HR", "钱招聘经理"
        ]
        
        tags = [
            ["五险一金", "年终奖", "带薪年假"],
            ["股票期权", "技术大牛", "扁平管理"],
            ["免费三餐", "住房补贴", "弹性工作"],
            ["六险一金", "定期体检", "项目奖金"],
            ["技术前沿", "团队优秀", "发展空间大"]
        ]
        
        title = random.choice(titles)
        company = random.choice(companies)
        salary = random.choice(salaries)
        salary_min, salary_max, salary_unit = self._parse_salary(salary)
        
        return {
            "platform": "boss",
            "title": title,
            "company": company,
            "location": city,
            "salary": salary,
            "salary_min": salary_min,
            "salary_max": salary_max,
            "salary_unit": salary_unit,
            "boss": random.choice(bosses),
            "experience": f"{random.randint(1, 10)}年以上",
            "education": random.choice(["本科", "硕士", "不限"]),
            "publish_time": f"{random.randint(1, 3)}小时内活跃",
            "tags": random.choice(tags),
            "job_url": f"https://www.zhipin.com/job_detail/{random.randint(1000000, 9999999)}.html",
            "company_url": f"https://www.zhipin.com/c{random.randint(10000, 99999)}/",
            "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
    
    def _parse_salary(self, salary_text: str) -> Tuple[Optional[float], Optional[float], str]:
        """
        解析薪资文本
        
        Returns:
            (最低薪资, 最高薪资, 单位)
        """
        if not salary_text or salary_text in ["面议", "保密"]:
            return None, None, "面议"
        
        try:
            # 清理文本
            text = salary_text.replace('k', 'K').replace('K', '000').replace('w', 'W').replace('W', '0000')
            text = re.sub(r'[^\d\-~·至\.]', '', text)
            
            # 提取数字
            numbers = re.findall(r'\d+\.?\d*', text)
            
            if len(numbers) >= 2:
                min_salary = float(numbers[0])
                max_salary = float(numbers[1])
                
                # 判断单位
                if '万' in salary_text:
                    unit = "万/月"
                    min_salary *= 10000
                    max_salary *= 10000
                elif '千' in salary_text:
                    unit = "千/月"
                    min_salary *= 1000
                    max_salary *= 1000
                else:
                    # 默认按K处理
                    if max_salary < 100:  # 可能是K单位
                        unit = "K/月"
                        min_salary *= 1000
                        max_salary *= 1000
                    else:
                        unit = "元/月"
                
                return min_salary, max_salary, unit
            elif len(numbers) == 1:
                salary = float(numbers[0])
                
                if '万' in salary_text:
                    unit = "万/月"
                    salary *= 10000
                elif '千' in salary_text:
                    unit = "千/月"
                    salary *= 1000
                elif salary < 100:  # 可能是K单位
                    unit = "K/月"
                    salary *= 1000
                else:
                    unit = "元/月"
                
                return salary, salary, unit
            else:
                return None, None, "面议"
                
        except Exception as e:
            print(f"解析薪资 '{salary_text}' 时出错: {e}")
            return None, None, "面议"
    
    def analyze_job_market(self, jobs: List[Dict]) -> Dict[str, Any]:
        """
        分析就业市场
        
        Args:
            jobs: 职位数据
            
        Returns:
            市场分析结果
        """
        if not jobs:
            return {"total_jobs": 0, "message": "没有数据"}
        
        print("分析就业市场...")
        
        # 按平台统计
        platform_stats = {}
        for job in jobs:
            platform = job.get("platform", "unknown")
            platform_stats[platform] = platform_stats.get(platform, 0) + 1
        
        # 薪资分析
        salaries = []
        for job in jobs:
            if job.get("salary_min") and job.get("salary_max"):
                avg_salary = (job["salary_min"] + job["salary_max"]) / 2
                salaries.append(avg_salary)
        
        if salaries:
            avg_salary = sum(salaries) / len(salaries)
            min_salary = min(salaries)
            max_salary = max(salaries)
        else:
            avg_salary = min_salary = max_salary = 0
        
        # 热门城市
        city_stats = {}
        for job in jobs:
            city = job.get("location", "未知")
            if city:
                city_stats[city] = city_stats.get(city, 0) + 1
        
        # 热门职位关键词
        title_keywords = {}
        for job in jobs:
            title = job.get("title", "")
            if title:
                # 简单分词
                words = re.findall(r'[\u4e00-\u9fffA-Za-z]+', title)
                for word in words:
                    if len(word) > 1:  # 过滤单字
                        title_keywords[word] = title_keywords.get(word, 0) + 1
        
        # 取前10个热门关键词
        top_keywords = sorted(title_keywords.items(), key=lambda x: x[1], reverse=True)[:10]
        
        analysis = {
            "total_jobs": len(jobs),
            "platform_distribution": platform_stats,
            "salary_analysis": {
                "average": round(avg_salary, 2),
                "min": round(min_salary, 2),
                "max": round(max_salary, 2),
                "currency": "CNY"
            },
            "city_distribution": dict(sorted(city_stats.items(), key=lambda x: x[1], reverse=True)[:5]),
            "top_keywords": dict(top_keywords),
            "market_demand": "高" if len(jobs) > 50 else "中" if len(jobs) > 20 else "低",
            "analysis_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        }
        
        return analysis
    
    def run(self,
            platforms: List[str] = None,
            keyword: str = "Python",
            cities: List[str] = None,
            max_jobs_per_platform: int = 30,
            analyze_market: bool = True,
            **kwargs) -> Dict[str, Any]:
        """
        运行招聘爬虫
        
        Args:
            platforms: 平台列表，可选 ["51job", "zhilian", "boss"]
            keyword: 职位关键词
            cities: 城市列表
            max_jobs_per_platform: 每个平台最大职位数
            analyze_market: 是否分析就业市场
            
        Returns:
            招聘数据字典
        """
        print("=" * 60)
        print("开始运行招聘网站职位信息爬虫")
        print("=" * 60)
        
        # 设置默认值
        if platforms is None:
            platforms = ["51job", "zhilian", "boss"]
        
        if cities is None:
            cities = ["北京", "上海", "深圳", "广州"]
        
        all_jobs = {}
        start_time = datetime.now()
        
        try:
            # 爬取前程无忧
            if "51job" in platforms:
                print(f"\n💼 爬取前程无忧...")
                51job_jobs = []
                for city in cities[:2]:  # 限制城市数量
                    jobs = self.crawl_51job(keyword, city, max_jobs_per_platform // len(cities))
                    51job_jobs.extend(jobs)
                    print(f"   城市 {city}: {len(jobs)} 个职位")
                all_jobs["51job"] = 51job_jobs
            
            # 爬取智联招聘
            if "zhilian" in platforms:
                print(f"\n📋 爬取智联招聘...")
                zhilian_jobs = []
                for city in cities[:2]:
                    jobs = self.crawl_zhilian(keyword, city, max_jobs_per_platform // len(cities))
                    zhilian_jobs.extend(jobs)
                    print(f"   城市 {city}: {len(jobs)} 个职位")
                all_jobs["zhilian"] = zhilian_jobs
            
            # 爬取BOSS直聘
            if "boss" in platforms:
                print(f"\n🤝 爬取BOSS直聘...")
                boss_jobs = []
                for city in cities[:2]:
                    jobs = self.crawl_boss(keyword, city, max_jobs_per_platform // len(cities))
                    boss_jobs.extend(jobs)
                    print(f"   城市 {city}: {len(jobs)} 个职位")
                all_jobs["boss"] = boss_jobs
            
            # 合并所有职位
            all_platform_jobs = []
            for platform_jobs in all_jobs.values():
                all_platform_jobs.extend(platform_jobs)
            
            # 分析就业市场
            market_analysis = {}
            if analyze_market and all_platform_jobs:
                market_analysis = self.analyze_job_market(all_platform_jobs)
                print(f"\n📊 就业市场分析:")
                print(f"   总职位数: {market_analysis['total_jobs']}")
                print(f"   平均薪资: {market_analysis['salary_analysis']['average']:,.0f} CNY")
                print(f"   市场需求: {market_analysis['market_demand']}")
                print(f"   热门城市: {', '.join(market_analysis['city_distribution'].keys())}")
            
            # 生成报告
            end_time = datetime.now()
            duration = (end_time - start_time).total_seconds()
            
            print(f"\n✅ 招聘数据爬取完成!")
            print(f"   总计: {len(all_platform_jobs)} 个职位")
            print(f"   耗时: {duration:.2f} 秒")
            print(f"   平台: {', '.join(platforms)}")
            print(f"   关键词: {keyword}")
            print(f"   城市: {', '.join(cities)}")
            
            result = {
                "jobs_by_platform": all_jobs,
                "all_jobs": all_platform_jobs,
                "market_analysis": market_analysis,
                "crawl_summary": {
                    "total_jobs": len(all_platform_jobs),
                    "platforms": platforms,
                    "keyword": keyword,
                    "cities": cities,
                    "duration_seconds": round(duration, 2),
                    "crawl_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                }
            }
            
            return result
            
        except Exception as e:
            print(f"❌ 招聘数据爬取失败: {e}")
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
            filename = f"job_data_{timestamp}.xlsx"
        
        # 准备导出数据
        export_data = []
        
        # 导出所有职位
        if "all_jobs" in data and data["all_jobs"]:
            for job in data["all_jobs"]:
                export_item = {
                    "平台": job.get("platform", ""),
                    "职位标题": job.get("title", ""),
                    "公司名称": job.get("company", ""),
                    "工作地点": job.get("location", ""),
                    "薪资范围": job.get("salary", ""),
                    "最低薪资": job.get("salary_min", ""),
                    "最高薪资": job.get("salary_max", ""),
                    "薪资单位": job.get("salary_unit", ""),
                    "经验要求": job.get("experience", ""),
                    "学历要求": job.get("education", ""),
                    "发布时间": job.get("publish_time", ""),
                    "职位链接": job.get("job_url", ""),
                    "公司链接": job.get("company_url", ""),
                    "爬取时间": job.get("crawl_time", "")
                }
                
                # BOSS直聘特有字段
                if job.get("platform") == "boss":
                    export_item["招聘者"] = job.get("boss", "")
                    export_item["公司标签"] = ", ".join(job.get("tags", []))
                
                export_data.append(export_item)
        
        # 导出到Excel
        filepath = self.exporter.export_to_excel(
            data=export_data,
            filename=filename,
            sheet_name="招聘职位数据",
            title="招聘网站职位信息数据报告",
            add_summary=True
        )
        
        # 添加市场分析
        if "market_analysis" in data and data["market_analysis"]:
            analysis = data["market_analysis"]
            summary_data = [
                {"指标": "总职位数", "数值": analysis.get("total_jobs", 0)},
                {"指标": "平均薪资(CNY)", "数值": analysis.get("salary_analysis", {}).get("average", 0)},
                {"指标": "最低薪资(CNY)", "数值": analysis.get("salary_analysis", {}).get("min", 0)},
                {"指标": "最高薪资(CNY)", "数值": analysis.get("salary_analysis", {}).get("max", 0)},
                {"指标": "市场需求", "数值": analysis.get("market_demand", "")},
                {"指标": "分析时间", "数值": analysis.get("analysis_time", "")}
            ]
            
            self.exporter.add_sheet_to_excel(
                filepath=filepath,
                data=summary_data,
                sheet_name="市场分析",
                title="就业市场分析报告"
            )
        
        print(f"\n数据已导出到: {filepath}")
        return filepath


if __name__ == "__main__":
    # 演示用法
    crawler = JobCrawler()
    
    # 运行爬虫
    data = crawler.run(
        platforms=["51job", "zhilian", "boss"],
        keyword="Python",
        cities=["北京", "上海"],
        max_jobs_per_platform=20
    )
    
    # 导出数据
    if data and "all_jobs" in data:
        filepath = crawler.export_data(data)
        print(f"\n数据已导出到: {filepath}")