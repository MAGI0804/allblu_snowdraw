from cate_method import *
from taobao_method import *
import requests
from urllib.parse import urlencode
from datetime import timedelta,datetime
import pymysql
import time
import json


def get_taobao_orders(fields, **kwargs):
    """
    获取淘宝订单数据

    :param app_key: 应用密钥
    :param app_secret: 应用密钥(用于签名)
    :param session: 用户会话令牌
    :param fields: 需要返回的字段列表
    :param kwargs: 其他可选参数
    :return: API响应结果
    """
    # 计算前一天的日期范围
    today = datetime.now()
    yesterday = today - timedelta(days=1)
    spec_day=today - timedelta(days=1)
    start_created = spec_day.replace(hour=0, minute=0, second=0).strftime("%Y-%m-%d %H:%M:%S")
    end_created = yesterday.replace(hour=23, minute=59, second=59).strftime("%Y-%m-%d %H:%M:%S")
    # start_created = today.replace(hour=0, minute=0, second=0).strftime("%Y-%m-%d %H:%M:%S")
    # end_created = today.replace(hour=23, minute=59, second=59).strftime("%Y-%m-%d %H:%M:%S")
    start_created = "2025-10-01 00:00:00"
    end_created = "2025-10-01 23:59:59"

    base_url = "http://gw.api.taobao.com/router/rest"
    page_no = 1  # 起始页码
    max_pages = 1000  # 设置最大页数限制

    all_data = []  # 存储所有页面的数据

    while page_no <= max_pages:
        # 准备基本参数（每次循环更新页码和时间戳）
        params = {
            "method": "taobao.trades.sold.get",
            "app_key": "34101613",
            "session": "6101607ec53f58e15fdbc10cc7baa1d2c3c4e292fce19183948263976",
            "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "v": "2.0",
            "sign_method": "md5",
            "format": "json",
            "start_created": start_created,
            "end_created": end_created,
            "page_no": page_no,
            "page_size": 100,
            "use_has_next": "true",
            "status": "TRADE_FINISHED",
            "fields": ",".join(fields)  # 将字段列表转换为逗号分隔字符串
        }

        # 添加额外参数
        params.update(kwargs)

        # 生成签名（假设generate_md5_signature函数已实现）
        params["sign"] = generate_md5_signature(params)  # 需要传入app_secret

        try:
            # 发送POST请求（参数在查询字符串）
            response = requests.post(f"{base_url}?{urlencode(params)}")
            response.raise_for_status()  # 检查HTTP错误
            data = response.json()

            # 处理当前页数据
            print(f"已获取完第 {page_no} 页数据")
            all_data.append(data)  # 保存当前页数据

            # 检查是否有下一页
            has_next = data.get("trades_sold_get_response", {}).get("has_next", False)
            if not has_next:
                print("已获取所有有效数据")
                break

            page_no += 1  # 准备获取下一页

        except Exception as e:
            print(f"请求第 {page_no} 页时出错: {str(e)}")
            break

    print(f"总共获取了 {len(all_data)} 页数据")
    return all_data  # 返回所有页面的数据


def extract_order_data(data_list):
    """
    从淘宝交易数据中提取tid, oid, pay_time, title, outer_iid, price, num, sku_properties_name, sign_time
    参数:
        data_list: 包含交易数据的列表
    返回:
        包含提取字段的字典列表
    """
    extracted_data = []

    # 遍历数据列表
    for data in data_list:
        trades_response = data.get('trades_sold_get_response', {})
        trades = trades_response.get('trades', {}).get('trade', [])

        # 遍历每个订单
        for trade in trades:
            tid = trade.get('tid', '')
            pay_time = trade.get('pay_time', '')
            oaid = trade.get('oaid', '')
            sign_time = trade.get('sign_time', '')  # 添加 sign_time 字段提取

            # 处理子订单
            orders = trade.get('orders', {}).get('order', [])
            # 统一处理为列表格式
            order_list = orders if isinstance(orders, list) else [orders]

            for order in order_list:
                extracted_data.append({
                    'tid': tid,
                    'oid': order.get('oid', ''),
                    'pay_time': pay_time,
                    'oaid': oaid,
                    'sign_time': sign_time,  # 添加 sign_time 到结果
                    'title': order.get('title', ''),
                    'num_iid': order.get('num_iid', ''),
                    'outer_iid': order.get('outer_iid', ''),
                    'price': order.get('price', ''),
                    'num': order.get('num', ''),
                    'cid': order.get('cid', ''),
                    'payment': order.get('payment', ''),
                    'divide_order_fee': order.get('divide_order_fee', ''),
                    'adjust_fee': order.get('adjust_fee', ''),
                    'discount_fee': order.get('discount_fee', ''),
                    'sku_properties_name': order.get('sku_properties_name', '')
                })
    return extracted_data

if __name__ == "__main__":
    result = get_taobao_orders(fields=["tid","created","status","payment","end_time", "pay_time","consign_time","sign_time","invoice_no","oaid"])

    extracted_data = extract_order_data(result)
    updated_data = replace_and_remove_cid(extracted_data)

    print(result)

