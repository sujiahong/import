import asyncio
from playwright.async_api import async_playwright
import pandas as pd

async def scrape_dianping(url):
    async with async_playwright() as p:
        # headless=False 会显示浏览器界面，方便手动处理登录和验证码
        browser = await p.chromium.launch(headless=False)
        context = await browser.new_context(
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        )
        page = await context.new_page()
        
        print(f"正在访问: {url}")
        await page.goto(url)
        
        # 留出 20 秒时间：页面打开后，请在弹出的浏览器中手动进行扫码登录或滑动验证码
        print("请在弹出的浏览器中处理验证码或扫码登录... 倒计时 20 秒")
        await page.wait_for_timeout(20000)
        
        shops_data = []
        
        # 获取店铺列表节点
        shop_elements = await page.query_selector_all('.shop-all-list li')
        print(f"找到 {len(shop_elements)} 家店铺，开始解析...")
        
        for shop in shop_elements:
            try:
                # 获取店名
                name_el = await shop.query_selector('.tit h4')
                name = await name_el.inner_text() if name_el else ""
                
                # 获取星级
                star_el = await shop.query_selector('.sml-rank-stars')
                star = await star_el.get_attribute('title') if star_el else ""
                
                # 获取评论数 (可能包含加密字体)
                review_el = await shop.query_selector('.review-num b')
                review = await review_el.inner_text() if review_el else ""
                
                # 获取人均价格 (可能包含加密字体)
                price_el = await shop.query_selector('.mean-price b')
                price = await price_el.inner_text() if price_el else ""
                
                # 获取商区/类型
                tag_els = await shop.query_selector_all('.tag')
                tags = [await t.inner_text() for t in tag_els] if tag_els else []
                shop_type = tags[0] if len(tags) > 0 else ""
                address = tags[1] if len(tags) > 1 else ""
                
                if name:
                    shops_data.append({
                        "店铺名称": name,
                        "星级": star,
                        "评论数": review,
                        "人均价格": price,
                        "类型": shop_type,
                        "商区": address
                    })
            except Exception as e:
                print(f"解析单条店铺报错: {e}")
                
        await browser.close()
        return shops_data

def save_to_excel(data, filename="dianping_shops.xlsx"):
    df = pd.DataFrame(data)
    df.to_excel(filename, index=False)
    print(f"\n成功！数据已保存至 Excel 文件: {filename}")

if __name__ == "__main__":
    # 以北京的美食频道为例
    target_url = "https://www.dianping.com/beijing/ch10"
    
    # 运行异步爬虫
    data = asyncio.run(scrape_dianping(target_url))
    
    if data:
        save_to_excel(data)
    else:
        print("\n未获取到数据，可能是因为被反爬拦截、未及时登录或页面DOM结构发生了变化。")