import requests
import json
import time

def get_douyin_hot_videos(min_popularity=1000000):
    # 抖音热榜接口
    # 注意：此接口可能会随时间变化，且通常需要有效的 Cookie 才能访问
    url = "https://www.douyin.com/aweme/v1/web/hot/search/list/"
    
    # 【重要】如果运行结果为空或报错，请在浏览器(F12->Network)登录抖音后复制 Cookie 填入下方
    cookies = {
        # 示例：'sessionid': 'xxx', 'ttwid': 'xxx'
        # 请替换为你自己的 Cookie
    }
    
    headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Referer': 'https://www.douyin.com/hot',
        'Accept': 'application/json, text/plain, */*',
    }

    try:
        # 尝试不带 Cookie 请求（可能会被重定向或返回空），如果失败则依赖用户填入的 Cookie
        response = requests.get(url, headers=headers, cookies=cookies, timeout=10)
        
        if response.status_code == 200:
            try:
                data = response.json()
            except json.JSONDecodeError:
                print("响应不是有效的 JSON 格式。")
                return []

            if 'data' in data and 'word_list' in data['data']:
                hot_list = data['data']['word_list']
                results = []
                
                for item in hot_list:
                    # hot_value 代表热度
                    hot_value = item.get('hot_value', 0)
                    
                    if hot_value >= min_popularity:
                        word = item.get('word', '')
                        # 热榜通常是“热词”，对应的链接一般是搜索结果页
                        video_link = f"https://www.douyin.com/search/{word}"
                        
                        results.append({
                            'title': word,
                            'hot_value': hot_value,
                            'link': video_link
                        })
                return results
            else:
                print("接口返回数据结构异常或未包含热榜数据。")
                # 打印部分响应以便调试
                # print(f"Debug Response: {json.dumps(data, ensure_ascii=False)[:200]}...")
        else:
            print(f"请求失败，状态码: {response.status_code}")
            
    except Exception as e:
        print(f"发生错误: {e}")
    
    return []

if __name__ == "__main__":
    threshold = 1000000
    print(f"正在获取热度超过 {threshold} 的抖音内容...")
    
    videos = get_douyin_hot_videos(threshold)
    
    if videos:
        print(f"\n找到 {len(videos)} 个热门内容：")
        print(f"{'标题':<30} | {'热度':<10} | {'链接'}")
        print("-" * 100)
        for v in videos:
            # 简单的格式化输出，截断过长的标题
            title = v['title']
            if len(title) > 28:
                title = title[:25] + "..."
            print(f"{title:<30} | {v['hot_value']:<10} | {v['link']}")
    else:
        print("\n[!] 未获取到数据。")
        print("原因可能是：")
        print("1. 抖音接口开启了反爬验证（需要有效的 Cookie）。")
        print("2. 网络连接问题。")
        print("建议：请打开 'python/get_douyin_hot.py' 文件，在 cookies 变量中填入浏览器获取的 Cookie。")
