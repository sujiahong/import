#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
主控制程序 - 管理和运行所有爬虫
"""

import argparse
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import List, Dict, Optional
import json

# 添加项目根目录到Python路径
project_root = Path(__file__).parent
sys.path.insert(0, str(project_root))

from utils.excel_exporter import ExcelExporter
from config.settings import (
    crawler_config,
    log_config,
    specific_config
)

# 导入所有爬虫
from crawlers.news.news_crawler import NewsCrawler
from crawlers.ecommerce.ecommerce_crawler import EcommerceCrawler
from crawlers.social_media.social_crawler import SocialMediaCrawler
from crawlers.job.job_crawler import JobCrawler
from crawlers.finance.finance_crawler import FinanceCrawler
from crawlers.government.government_crawler import GovernmentCrawler
from crawlers.real_estate.real_estate_crawler import RealEstateCrawler
from crawlers.academic.academic_crawler import AcademicCrawler
from crawlers.travel.travel_crawler import TravelCrawler
from crawlers.video.video_crawler import VideoCrawler


class CrawlerManager:
    """爬虫管理器"""
    
    def __init__(self, output_dir: Optional[Path] = None):
        """
        初始化爬虫管理器
        
        Args:
            output_dir: 输出目录
        """
        self.output_dir = output_dir or project_root / "output"
        self.output_dir.mkdir(exist_ok=True)
        
        self.crawlers = {}
        self.exporter = ExcelExporter(self.output_dir)
        
        # 注册所有爬虫
        self._register_crawlers()
    
    def _register_crawlers(self):
        """注册所有爬虫"""
        # 新闻爬虫
        self.crawlers["news"] = {
            "name": "新闻资讯聚合爬虫",
            "class": NewsCrawler,
            "description": "爬取新浪、腾讯、网易、头条等新闻网站",
            "config": specific_config.NEWS
        }
        
        # 电商爬虫
        self.crawlers["ecommerce"] = {
            "name": "电商价格监控爬虫",
            "class": EcommerceCrawler,
            "description": "监控淘宝、京东、拼多多、亚马逊等电商平台价格",
            "config": specific_config.ECOMMERCE
        }
        
        # 社交媒体爬虫
        self.crawlers["social_media"] = {
            "name": "社交媒体情感分析爬虫",
            "class": SocialMediaCrawler,
            "description": "爬取微博、知乎、小红书等社交媒体内容",
            "config": specific_config.SOCIAL_MEDIA
        }
        
        # 招聘网站爬虫
        self.crawlers["job"] = {
            "name": "招聘网站职位信息爬虫",
            "class": JobCrawler,
            "description": "收集前程无忧、智联招聘等招聘网站信息",
            "config": specific_config.JOB
        }
        
        # 房地产爬虫
        self.crawlers["real_estate"] = {
            "name": "房地产市场价格爬虫",
            "class": RealEstateCrawler,
            "description": "监控链家、贝壳等房地产平台价格",
            "config": specific_config.REAL_ESTATE
        }
        
        # 金融数据爬虫
        self.crawlers["finance"] = {
            "name": "股票金融数据爬虫",
            "class": FinanceCrawler,
            "description": "获取东方财富、新浪财经等金融数据",
            "config": specific_config.FINANCE
        }
        
        # 学术论文爬虫
        self.crawlers["academic"] = {
            "name": "学术论文文献爬虫",
            "class": AcademicCrawler,
            "description": "爬取知网、万方等学术数据库论文",
            "config": specific_config.ACADEMIC
        }
        
        # 旅游网站爬虫
        self.crawlers["travel"] = {
            "name": "旅游网站酒店价格爬虫",
            "class": TravelCrawler,
            "description": "监控携程、去哪儿等旅游平台价格",
            "config": specific_config.TRAVEL
        }
        
        # 视频平台爬虫
        self.crawlers["video"] = {
            "name": "视频平台热门内容爬虫",
            "class": VideoCrawler,
            "description": "分析B站、抖音等视频平台热门内容",
            "config": specific_config.VIDEO
        }
        
        # 政府数据爬虫
        self.crawlers["government"] = {
            "name": "政府公开数据爬虫",
            "class": GovernmentCrawler,
            "description": "收集政府数据开放平台公开数据",
            "config": specific_config.GOVERNMENT
        }
    
    def list_crawlers(self) -> List[Dict]:
        """列出所有可用的爬虫"""
        crawler_list = []
        
        for key, info in self.crawlers.items():
            status = "可用" if info["class"] else "待开发"
            crawler_list.append({
                "id": key,
                "name": info["name"],
                "description": info["description"],
                "status": status,
                "config": info["config"]
            })
        
        return crawler_list
    
    def get_crawler_info(self, crawler_id: str) -> Optional[Dict]:
        """获取特定爬虫的详细信息"""
        if crawler_id not in self.crawlers:
            return None
        
        info = self.crawlers[crawler_id]
        return {
            "id": crawler_id,
            "name": info["name"],
            "description": info["description"],
            "class_available": info["class"] is not None,
            "config": info["config"]
        }
    
    def run_crawler(self, crawler_id: str, **kwargs) -> Dict:
        """
        运行指定爬虫
        
        Args:
            crawler_id: 爬虫ID
            **kwargs: 爬虫参数
            
        Returns:
            运行结果
        """
        if crawler_id not in self.crawlers:
            return {
                "success": False,
                "error": f"爬虫 '{crawler_id}' 不存在",
                "data": None,
                "stats": None
            }
        
        info = self.crawlers[crawler_id]
        
        if info["class"] is None:
            return {
                "success": False,
                "error": f"爬虫 '{crawler_id}' 尚未开发完成",
                "data": None,
                "stats": None
            }
        
        print(f"开始运行爬虫: {info['name']}")
        print(f"描述: {info['description']}")
        print("-" * 50)
        
        try:
            # 创建爬虫实例
            crawler = info["class"]()
            
            # 运行爬虫
            start_time = time.time()
            data = crawler.run(**kwargs)
            end_time = time.time()
            
            # 获取统计信息
            crawler_stats = crawler.get_stats()
            crawler_stats["total_time_seconds"] = end_time - start_time
            
            # 导出Excel
            excel_file = self.exporter.export_crawler_data(
                crawler_name=crawler_id,
                data=data,
                crawler_stats=crawler_stats
            )
            
            result = {
                "success": True,
                "crawler_name": info["name"],
                "data_count": len(data),
                "excel_file": str(excel_file),
                "stats": crawler_stats,
                "timestamp": datetime.now().isoformat()
            }
            
            print(f"\n爬虫运行完成!")
            print(f"爬取数据: {len(data)} 条")
            print(f"Excel文件: {excel_file}")
            print(f"总耗时: {end_time - start_time:.2f} 秒")
            
            return result
            
        except Exception as e:
            print(f"爬虫运行失败: {e}")
            return {
                "success": False,
                "error": str(e),
                "data": None,
                "stats": None
            }
    
    def run_all_crawlers(self) -> Dict[str, Dict]:
        """
        运行所有已开发的爬虫
        
        Returns:
            所有爬虫的运行结果
        """
        results = {}
        
        for crawler_id, info in self.crawlers.items():
            if info["class"] is not None:
                print(f"\n{'='*60}")
                print(f"运行爬虫: {crawler_id}")
                print(f"{'='*60}")
                
                result = self.run_crawler(crawler_id)
                results[crawler_id] = result
                
                # 延迟避免同时请求过多
                time.sleep(2)
        
        return results
    
    def generate_report(self, results: Dict[str, Dict]) -> Path:
        """
        生成运行报告
        
        Args:
            results: 爬虫运行结果
            
        Returns:
            报告文件路径
        """
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = self.output_dir / f"crawler_report_{timestamp}.json"
        
        # 汇总统计
        summary = {
            "total_crawlers": len(results),
            "successful_crawlers": sum(1 for r in results.values() if r["success"]),
            "failed_crawlers": sum(1 for r in results.values() if not r["success"]),
            "total_data_items": sum(r.get("data_count", 0) for r in results.values() if r["success"]),
            "run_timestamp": datetime.now().isoformat(),
            "results": results
        }
        
        # 保存报告
        with open(report_file, 'w', encoding='utf-8') as f:
            json.dump(summary, f, ensure_ascii=False, indent=2)
        
        print(f"\n运行报告已生成: {report_file}")
        print(f"总爬虫数: {summary['total_crawlers']}")
        print(f"成功爬虫: {summary['successful_crawlers']}")
        print(f"失败爬虫: {summary['failed_crawlers']}")
        print(f"总数据项: {summary['total_data_items']}")
        
        return report_file


def print_crawler_list(manager: CrawlerManager):
    """打印爬虫列表"""
    crawlers = manager.list_crawlers()
    
    print("可用爬虫列表:")
    print("-" * 80)
    
    for crawler in crawlers:
        status_icon = "✓" if crawler["status"] == "可用" else "⏳"
        print(f"{status_icon} [{crawler['id']:12s}] {crawler['name']}")
        print(f"   描述: {crawler['description']}")
        
        # 显示额外信息
        if crawler['id'] == 'ecommerce':
            print("   ⚠️  注意: 需要Chrome浏览器和ChromeDriver")
            print("   ⚠️  注意: 可能需要VPN访问某些网站")
        
        print()



def demo_news_crawler():
    """演示新闻爬虫"""
    print("演示: 运行新闻资讯聚合爬虫")
    print("-" * 50)
    
    manager = CrawlerManager()
    
    result = manager.run_crawler(
        "news",
        sources=["sina", "tencent"],
        categories=["technology"],
        max_articles=5
    )
    
    return result



def demo_ecommerce_crawler():
    """演示电商爬虫"""
    print("演示: 运行电商价格监控爬虫")
    print("-" * 50)
    print("注意: 由于电商网站反爬虫严格，此演示使用模拟数据")
    print("实际使用时请确保已安装Chrome浏览器和ChromeDriver")
    print("-" * 50)
    
    from crawlers.ecommerce.ecommerce_crawler import demo_ecommerce_crawler as real_demo
    result = real_demo()
    
    return result


def main():
    """主函数"""
    parser = argparse.ArgumentParser(
        description="爬虫管理系统 - 运行和管理各种爬虫程序",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    
    parser.add_argument(
        "--list", 
        action="store_true",
        help="列出所有可用的爬虫"
    )
    
    parser.add_argument(
        "--info", 
        metavar="CRAWLER_ID",
        help="显示指定爬虫的详细信息"
    )
    
    parser.add_argument(
        "--run", 
        metavar="CRAWLER_ID",
        help="运行指定的爬虫"
    )
    
    parser.add_argument(
        "--run-all", 
        action="store_true",
        help="运行所有已开发的爬虫"
    )
    
    parser.add_argument(
        "--demo", 
        action="store_true",
        help="运行演示程序（新闻爬虫）"
        action="store_true",
        help="运行演示程序（新闻爬虫）"
    )
    
    parser.add_argument(
        "--output-dir", 
        metavar="DIR",
        help="指定输出目录"
    )
    
    parser.add_argument(
        "--demo-ecommerce", 
        action="store_true",
        help="运行电商爬虫演示程序"
    )
    
    args = parser.parse_args()
    
    # 设置输出目录
    output_dir = None
    if args.output_dir:
        output_dir = Path(args.output_dir)
    
    # 创建管理器
    manager = CrawlerManager(output_dir)
    
    # 处理命令行参数
    if args.list:
        print_crawler_list(manager)
    
    elif args.info:
        info = manager.get_crawler_info(args.info)
        if info:
            print(f"\n爬虫信息: {args.info}")
            print("-" * 50)
            print(f"名称: {info['name']}")
            print(f"描述: {info['description']}")
            print(f"状态: {'可用' if info['class_available'] else '待开发'}")
            print(f"\n配置:")
            for key, value in info['config'].items():
                print(f"  {key}: {value}")
        else:
            print(f"错误: 爬虫 '{args.info}' 不存在")
    
    elif args.run:
        result = manager.run_crawler(args.run)
        if not result["success"]:
            print(f"运行失败: {result.get('error', '未知错误')}")
    
    elif args.run_all:
        print("开始运行所有已开发的爬虫...")
        results = manager.run_all_crawlers()
        
        # 生成报告
        report_file = manager.generate_report(results)
        print(f"详细报告已保存到: {report_file}")
    
    elif args.demo_ecommerce:
        result = demo_ecommerce_crawler()
        if result:
            print(f"\n演示完成!")
            print(f"生成 {len(result)} 个模拟商品")
        else:
            print(f"演示失败")
        elif args.demo:
        result = demo_news_crawler()
        if result["success"]:
            print(f"\n演示完成!")
            print(f"Excel文件: {result['excel_file']}")
        else:
            print(f"演示失败: {result.get('error', '未知错误')}")
    
    else:
        # 没有参数时显示帮助信息
        parser.print_help()
        print("\n示例:")
        print("  python main.py --list                    # 列出所有爬虫")
        print("  python main.py --info news              # 查看新闻爬虫信息")
        print("  python main.py --run news               # 运行新闻爬虫")
        print("  python main.py --demo                   # 运行演示程序")
        print("  python main.py --run-all                # 运行所有已开发爬虫")


if __name__ == "__main__":
    print(f"爬虫管理系统 v1.0")
    print(f"当前时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"项目目录: {project_root}")
    print()
    
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n程序被用户中断")
        sys.exit(0)
    except Exception as e:
        print(f"\n程序运行出错: {e}")
        sys.exit(1)