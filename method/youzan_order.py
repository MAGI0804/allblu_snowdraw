import json
from datetime import datetime, timedelta
import requests


def get_youzan_access_token():
    """获取有赞open平台access_token"""
    url = "https://open.youzanyun.com/auth/token"
    headers = {"Content-Type": "application/json"}
    data = {
        "authorize_type": "silent",
        "client_id": "379981eff640bbb278",
        "client_secret": "1ef6d04d42b03784bd75fc1b74493c06",
        "grant_id": "15707004",
        "refresh": False
    }

    try:
        response = requests.post(url, json=data, headers=headers)
        response.raise_for_status()
        result = response.json()

        if result.get("success") and result.get("code") == 200:
            return result["data"]["access_token"]
        print(f"获取access_token失败: {result.get('message')}")
        return None
    except Exception as e:
        print(f"请求异常: {str(e)}")
        return None


# def get_store_orders(access_token, target_store):
#     """
#     获取指定店铺的完整订单信息
#     参数：
#     access_token - 有赞API访问令牌
#     target_store - 要筛选的目标店铺名称
#     """
#     url = "https://open.youzanyun.com/api/youzan.trades.sold.get/4.0.4"
#     headers = {"Content-Type": "application/json"}
#
#     # 设置当天时间范围
#     today = datetime.now().strftime("%Y-%m-%d")
#     params = {
#         "page_size": "100",
#         "status": "TRADE_SUCCESS",
#         "start_success": f"{today} 00:00:00",
#         "end_success": f"{today} 23:59:59"
#     }
#     # print(params)
#
#     try:
#         response = requests.post(
#             f"{url}?access_token={access_token}",
#             json=params,
#             headers=headers
#         )
#         response.raise_for_status()
#         result = response.json()
#
#         if not result.get("success"):
#             print(f"API请求失败: {result.get('message')}")
#             return []
#         # print(result)
#
#         # 筛选目标店铺订单
#         filtered_orders = []
#         for order in result["data"]["full_order_info_list"]:
#             order_info = order["full_order_info"]["order_info"]
#             if order_info.get("shop_name") == target_store:
#                 print(order_info.get("shop_name"))
#                 filtered_orders.append(order["full_order_info"])
#
#         return filtered_orders
#     except Exception as e:
#         print(f"订单获取异常: {str(e)}")
#         return []

def get_store_orders(access_token):
    """
    获取指定店铺的完整订单信息（按小时分段请求避免分页限制）
    参数：
    access_token - 有赞API访问令牌
    target_store - 要筛选的目标店铺名称
    """
    url = "https://open.youzanyun.com/api/youzan.trades.sold.get/4.0.4"
    headers = {"Content-Type": "application/json"}
    all_orders = []  # 存储所有订单

    # 获取当天日期
    today = datetime.now().strftime("%Y-%m-%d")
    print(today)
    # today ="2025-07-08"

    try: 
        # 循环24小时段
        for hour in range(23):
            # 计算当前小时的开始和结束时间
            start_time = f"{today} {hour:02d}:00:00"
            # start_time = f"2024-06-08 {hour:02d}:00:00"
            end_hour = (hour + 1) % 24
            end_date = today if end_hour != 0 else (datetime.now() + timedelta(days=1)).strftime("%Y-%m-%d")
            end_time = f"{end_date} {end_hour:02d}:00:00"

            next_cursor = None  # 初始化分页游标
            has_next_page = True

            # 当前小时段的分页循环
            while has_next_page:
                params = {
                    "page_size": "100",
                    "start_success": start_time,
                    "end_success": end_time
                }

                # 添加分页游标参数
                if next_cursor:
                    params["cursor"] = next_cursor

                response = requests.post(
                    f"{url}?access_token={access_token}",
                    json=params,
                    headers=headers
                )
                response.raise_for_status()
                result = response.json()

                if not result.get("success"):
                    # print(str(result))
                    print(f"API请求失败: {result.get('message')}")
                    print(result)
                    break

                # 处理当前页数据
                if "data" in result and "full_order_info_list" in result["data"]:
                    for order in result["data"]["full_order_info_list"]:
                            all_orders.append(order["full_order_info"])

                # 检查分页信息
                if "data" in result and "paginator" in result["data"]:
                    has_next_page = result["data"]["paginator"].get("has_next", False)
                    next_cursor = result["data"]["paginator"].get("next_cursor")
                else:
                    has_next_page = False
        # print(all_orders)

        return all_orders

    except Exception as e:
        print(f"订单获取异常: {str(e)}")
        return []




# 使用示例
if __name__ == "__main__":
    # 获取访问令牌和订单数据（接前文代码）
    token = get_youzan_access_token()
    print(token)
    orders = get_store_orders(token)
    print(orders)
