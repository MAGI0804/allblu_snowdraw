import requests
import time
import hashlib
import datetime
from pdd_sign import *


def get_pdd_orders(access_token, client_secret):
    """
    获取拼多多订单列表（简化版数据结构）

    Args:
        access_token (str or dict): 访问令牌，可以是单个token或店铺token字典
        client_secret (str): 应用密钥

    Returns:
        list: 所有店铺的订单列表，每个订单包含店铺名和关键字段
    """
    # 固定的client_id
    CLIENT_ID = "4b415953a5294085b1559afc0c453cb7"

    # 计算昨天的时间范围
    today = datetime.date.today()
    yesterday = today - datetime.timedelta(days=1)

    # 定义四个时间段
    time_periods = [
        {"name": "凌晨", "start": 0, "end": 6},  # 00:00-06:00
        {"name": "上午", "start": 6, "end": 12},  # 06:00-12:00
        {"name": "下午", "start": 12, "end": 18},  # 12:00-18:00
        {"name": "晚上", "start": 18, "end": 0}  # 18:00-24:00 (第二天0点)
    ]

    # 如果传入的是单个token，转换为字典格式
    if isinstance(access_token, str):
        access_tokens = {"默认店铺": access_token}
    else:
        access_tokens = access_token

    all_orders = []

    for shop_name, token in access_tokens.items():
        print(f"正在获取 {shop_name} 的订单...")

        # 按时间段获取订单
        for period in time_periods:
            period_name = period["name"]
            print(f"  获取 {period_name} 时间段订单...")

            # 计算时间段的时间戳
            start_time = datetime.datetime.combine(yesterday, datetime.time(period["start"], 0, 0))

            # 对于结束时间为0的情况，表示是第二天的0点
            if period["end"] == 0:
                end_time = datetime.datetime.combine(yesterday + datetime.timedelta(days=1),
                                                     datetime.time(0, 0, 0)) - datetime.timedelta(seconds=1)
            else:
                end_time = datetime.datetime.combine(yesterday,
                                                     datetime.time(period["end"], 0, 0)) - datetime.timedelta(seconds=1)

            start_confirm_at = int(start_time.timestamp())
            end_confirm_at = int(end_time.timestamp())

            page = 1
            page_size = 100
            max_pages = 10

            # 分页获取订单
            while page <= max_pages:
                # 准备请求参数
                timestamp = str(int(time.time()))

                params = {
                    "type": "pdd.order.list.get",
                    "timestamp": timestamp,
                    "client_id": CLIENT_ID,
                    "data_type": "json",
                    "access_token": token,
                    "page": str(page),
                    "page_size": str(page_size),
                    "order_status": "5",
                    "refund_status": "5",
                    "start_confirm_at": str(start_confirm_at),
                    "end_confirm_at": str(end_confirm_at)
                }

                # 生成签名
                sign = generate_sign_advanced(params, client_secret)
                params["sign"] = sign

                try:
                    # 发送POST请求
                    url = "https://gw-api.pinduoduo.com/api/router"
                    response = requests.post(url, data=params)
                    response.raise_for_status()

                    # 解析响应
                    result = response.json()

                    # 检查是否有错误
                    if "error_response" in result:
                        error_msg = result["error_response"]
                        print(f"    获取订单失败: {error_msg.get('error_msg', '未知错误')}")
                        break

                    # 处理订单数据
                    if "order_list_get_response" in result:
                        order_data = result["order_list_get_response"]
                        orders = order_data.get('order_list', [])

                        if orders:
                            # 提取关键字段并添加店铺名
                            for order in orders:
                                simplified_order = {
                                    "shop_name": shop_name,
                                    "order_sn": order.get("order_sn"),
                                    "pay_amount": order.get("pay_amount"),
                                    "pay_time": order.get("pay_time"),
                                    "item_list": []
                                }

                                # 处理商品列表
                                item_list = order.get("item_list", [])
                                for item in item_list:
                                    simplified_item = {
                                        "goods_count": item.get("goods_count"),
                                        "goods_id": item.get("goods_id"),
                                        "goods_name": item.get("goods_name"),
                                        "goods_price": item.get("goods_price"),
                                        "outer_goods_id": item.get("outer_goods_id"),
                                        "outer_id": item.get("outer_id"),
                                        "goods_spec": item.get("goods_spec"),
                                        "sku_id": item.get("sku_id")
                                    }
                                    simplified_order["item_list"].append(simplified_item)

                                all_orders.append(simplified_order)

                            print(f"    获取到 {len(orders)} 条订单")

                            # 如果返回的订单数量小于page_size，说明已经是最后一页
                            if len(orders) < page_size:
                                break
                        else:
                            # 没有更多订单
                            break
                    else:
                        print("    响应格式不正确")
                        break

                except requests.exceptions.RequestException as e:
                    print(f"    网络请求失败: {str(e)}")
                    break
                except ValueError as e:
                    print(f"    响应解析失败: {str(e)}")
                    break

                page += 1
                # 添加延迟避免请求过于频繁
                time.sleep(0.5)

            print(f"  {period_name}时间段获取完成")

        print(f"{shop_name} 订单获取完成，共获取 {len([o for o in all_orders if o['shop_name'] == shop_name])} 条订单\n")

    return all_orders


# 主函数示例
if __name__ == "__main__":
    # 应用密钥
    CLIENT_SECRET = "c584c4924f5ed15e393f1f16cb30993c12a655ad"

    # 店铺access_token映射
    access_tokens = {
        "拼多多官方旗舰店": "3bab67a2267e469b8af8c650f3f01c1f0f68d26b",
        "拼多多童装旗舰店": "94049dfee30044b7ac5632bbe0163ff3480e0199",
        "拼多多户外旗舰店": "4b143a73fc2346eaa226c10672b05377905087df"
     }

    # 获取所有店铺订单
    all_orders = get_pdd_orders(access_tokens, CLIENT_SECRET)

    # 打印汇总信息

    print(all_orders)



