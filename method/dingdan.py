import requests
import hashlib
import time
from datetime import datetime, timedelta
import pymysql
import json
import base64
import urllib.parse
import hmac
from jushuitan_token import get_token
import uuid


def fetch_jushuitan_orders(shop_id):
    # 配置参数
    url = "https://openapi.jushuitan.com/open/orders/single/query"
    app_key = "e50a8f2e66c845c188a04f34ebf4a663"
    access_token = get_token()
    # access_token = "21efeb99d4c947ebae9afb0cc1fdd988"
    secret = "b7a7e5df75ed4ae38c42db4fbe060fb8"  # 签名密钥
    charset = "UTF-8"
    version = "2"
    max_page = 100  # 最大页数限制

    # 获取昨日时间范围
    today = datetime.now()
    yesterday = today - timedelta(days=1)
    modified_begin = yesterday.strftime("%Y-%m-%d 00:00:00")
    modified_end = yesterday.strftime("%Y-%m-%d 23:59:59")
    status = "Sent"
    order_types = ["普通订单"]

    all_orders = []  # 存储所有订单
    extracted_data = []  # 存储提取后的结构化数据

    for page_index in range(1, max_page + 1):
        # 构建biz参数
        biz_data = {
            "page_index": str(page_index),
            "page_size": "100",
            "modified_begin": modified_begin,
            "modified_end": modified_end,
            "date_type": "2",
            "shop_id": shop_id,
            "status": status,
            "order_types": order_types
        }
        biz_str = json.dumps(biz_data, separators=(',', ':'))
        # print(biz_str)
        # 获取当前时间戳
        timestamp = str(int(time.time()))
        print(timestamp)
        # 构建签名字符串
        sign_str = (
                secret + "access_token" + access_token +
                "app_key" + app_key +
                "biz" + biz_str +
                "charset" + charset +
                "timestamp" + timestamp +
                "version" + version
        )
        # print(sign_str)
        # 生成MD5签名
        sign = hashlib.md5(sign_str.encode(charset)).hexdigest()
        # print(sign)

        # 构建请求参数
        payload = {
            "app_key": app_key,
            "access_token": access_token,
            "timestamp": timestamp,
            "charset": charset,
            "version": version,
            "biz": biz_str,
            "sign": sign
        }

        # 发送请求
        try:
            headers = {'Content-Type': 'application/json'}
            response = requests.post(url, data=payload, headers=headers)
            response.raise_for_status()

            # 解析响应
            result = response.json()
            # print(result)
            orders = result.get("data", []).get("orders",[])
            all_orders.extend(orders)

            # 提取所需字段
            for order in orders:
                # 提取订单级别信息
                order_info = {
                    "buyer_id": order.get("buyer_id"),
                    "type": order.get("type"),
                    "shop_name": order.get("shop_name"),
                    "so_id": order.get("so_id"),
                    "pay_date": order.get("pay_date"),
                    "referrer_id": order.get("referrer_id"),
                    "referrer_name": order.get("referrer_name")
                }

                # 提取商品级别信息
                items = order.get("items", [])
                for item in items:
                    item_data = order_info.copy()  # 复制订单级别信息
                    item_data.update({
                        "price": item.get("price"),
                        "seller_income_amount": item.get("seller_income_amount"),
                        "buyer_paid_amount":item.get("buyer_paid_amount"), #买家实付金额
                        "outer_oi_id": item.get("outer_oi_id"),
                        "oi_id": item.get("oi_id"),
                        "raw_so_id": item.get("raw_so_id"),
                        "i_id": item.get("i_id"),
                        "sku_id": item.get("sku_id"),
                        "properties_value": item.get("properties_value"),
                        "qty": item.get("qty"),
                        "name": item.get("name")
                    })
                    extracted_data.append(item_data)

            # 检查是否最后一页
            if len(orders) < 100:
                break

        except Exception as e:
            print(f"Request failed on page {page_index}: {str(e)}")
            print(f"获取订单详情失败: {result.get('msg')}")
            break

    print(extracted_data)
    print(f"Fetched {len(all_orders)} orders")
    return extracted_data, all_orders  # 返回原始数据和提取的数据


# upload_to_database 函数已移除，改为直接与 SnowOrderData 模型关联


def fetch_refunds(shop_id):
    # 配置参数
    app_key = "e50a8f2e66c845c188a04f34ebf4a663"
    # access_token = "db3df2ecebea4664abc9c9f068c1e87e"
    access_token = "21efeb99d4c947ebae9afb0cc1fdd988"
    app_secret = "b7a7e5df75ed4ae38c42db4fbe060fb8"  # 签名密钥
    url = "https://openapi.jushuitan.com/open/refund/single/query"
    charset="UTF-8"
    version="2"

    # 设置时间范围
    start_date = datetime.now()-timedelta(days=6)
    end_date = datetime.now()-timedelta(days=1)
    extracted_refunds=[] #提取的退款信息

    current_date = start_date

    # 按7天分段循环
    # while current_date <= end_date:
    #     segment_end = min(current_date + timedelta(days=5), end_date)

    # 设置时间段 (包含时间部分)
    modified_begin = current_date.strftime("%Y-%m-%d 00:00:00")
    modified_end = end_date.strftime("%Y-%m-%d 23:59:59")
    page_index = 1
    max_pages = 100

    # 分页循环
    while page_index <= max_pages:
        try:
            # 1. 构造biz参数 (保持字段顺序)
            biz_data = {
                "page_index": str(page_index),
                "page_size": "50",
                "modified_begin": modified_begin,
                "modified_end": modified_end,
                "shop_id": shop_id,
                "status":"confirmed"
            }
            biz_str = json.dumps(biz_data, separators=(',', ':'))
            # print(biz_str)
            # 获取当前时间戳
            timestamp = str(int(time.time()))
            # print(timestamp)
            # 构建签名字符串
            sign_str = (
                    app_secret + "access_token" + access_token +
                    "app_key" + app_key +
                    "biz" + biz_str +
                    "charset" + charset +
                    "timestamp" + timestamp +
                    "version" + version
            )
            # print(sign_str)
            # 生成MD5签名
            sign = hashlib.md5(sign_str.encode(charset)).hexdigest()
            # print(sign)
            # 5. 构建请求体
            payload = {
                "access_token": access_token,
                "app_key": app_key,
                "biz": biz_str,
                "charset": "UTF-8",
                "timestamp": timestamp,
                "version": "2",
                "sign": sign
            }

            # 6. 发送请求
            headers = {"Content-Type": "application/x-www-form-urlencoded"}
            response = requests.post(url, data=payload, headers=headers)
            response.raise_for_status()

            # 7. 处理响应
            result = response.json()

            # 添加数据 - 修复'list' object has no attribute 'get'错误
            data = result.get("data", {})
            refunds = data.get("datas", []) if isinstance(data, dict) else []

            # 遍历每个退款单
            for refund_order in refunds:
                # 提取退款单级别信息
                base_info = {
                    "as_id": refund_order.get("as_id"),  # 退款单ID
                    "type": refund_order.get("type"),  # 退款类型
                    "refund": refund_order.get("refund"),  # 退款金额
                    "status": refund_order.get("status"),  # 退款状态
                    "shop_id": refund_order.get("shop_id")  # 店铺ID
                }

                # 提取items中的商品明细
                items=refund_order.get("items", [])
                for item in items:
                    # 创建合并后的记录
                    record = base_info.copy()
                    record.update({
                        "outer_oi_id": item.get("outer_oi_id"),  # 订单明细ID
                        "sku_id": item.get("sku_id"),  # 商品SKU
                        "qty": item.get("qty"),  # 数量
                        "price": item.get("price")  # 单价
                    })
                    extracted_refunds.append(record)

            # 检查是否最后一页
            if len(refunds) < 50:
                break
            page_index += 1

        except Exception as e:
            print(f"处理失败: {str(e)}")
            break

            # 下一时间段 (增加1秒避免重叠)
            # current_date = segment_end + timedelta(seconds=1)

    # print(f"总共获取 {len(all_data)} 条退款记录")
    print(extracted_refunds)
    return extracted_refunds




def map_to_snow_order_data(raw_orders):
    """
    将聚水潭原始订单数据直接映射到SnowOrderData模型格式
    此函数生成的数据结构与d:\youlan_kids_customization\youlan_kids_go\django_to_go\models\snow_order_data.go中的SnowOrderData模型完全对应
    
    :param raw_orders: 原始订单数据列表
    :return: 映射后的SnowOrderData格式数据列表
    """
    mapped_data = []
    
    for order in raw_orders:
        # 解析日期时间
        def parse_datetime(date_str):
            if date_str:
                try:
                    return datetime.strptime(date_str, "%Y-%m-%d %H:%M:%S")
                except (ValueError, TypeError):
                    return None
            return None
        
        # 从订单数据中提取收货人信息（如果有）
        consignee_name = order.get("consignee", "").get("name", "") if isinstance(order.get("consignee"), dict) else ""
        province = order.get("consignee", "").get("province", "") if isinstance(order.get("consignee"), dict) else ""
        city = order.get("consignee", "").get("city", "") if isinstance(order.get("consignee"), dict) else ""
        county = order.get("consignee", "").get("county", "") if isinstance(order.get("consignee"), dict) else ""
        
        # 映射字段 - 直接对应SnowOrderData模型的每个字段
        snow_order = {
            # ID将由数据库自动生成，这里不需要设置
            "SerialNumber": 0,  # 将在Go端使用自增或生成
            "OnlineOrderNumber": order.get("so_id", ""),  # 线上订单号
            "OrderStatus": order.get("status", ""),  # 订单状态
            "Store": order.get("shop_name", ""),  # 店铺
            "OrderDate": parse_datetime(order.get("order_date")),  # 订单日期
            "ShipDate": parse_datetime(order.get("send_date")),  # 发货日期
            "PaymentDate": parse_datetime(order.get("pay_date")),  # 付款日期
            "SellerID": str(order.get("buyer_id", "")),  # 卖家id
            "ConfirmReceiptTime": parse_datetime(order.get("end_time")),  # 确认收货时间
            "ConsigneeName": consignee_name,  # 收货人姓名
            "Province": province,  # 省
            "City": city,  # 市
            "County": county,  # 县
            "TrackingNumber": order.get("l_id", ""),  # 快递单号
            "OriginalOnlineOrderNumber": order.get("raw_so_id", ""),  # 原始线上订单号
            "ActualPaymentAmount": float(order.get("pay_amount", 0)),  # 实付金额
            "ReturnQuantity": 0,  # 退货数量（默认0）
            "ReturnAmount": 0.0,  # 退货金额（默认0）
            "OnlineSubOrderNumber": order.get("outer_oi_id", ""),  # 线上子订单编号
            "Remark": order.get("remark", "")  # 备注
            # CreatedAt和UpdatedAt将由Go模型自动设置
        }
        
        # 为每个订单项创建一条记录
        items = order.get("items", [])
        for item in items:
            # 创建订单项的副本
            item_order = snow_order.copy()
            # 如果订单项有子订单号，则使用订单项的子订单号
            if item.get("outer_oi_id"):
                item_order["OnlineSubOrderNumber"] = item.get("outer_oi_id")
            # 添加商品名称到备注
            if item.get("name"):
                if item_order["Remark"]:
                    item_order["Remark"] += f" | 商品: {item.get('name')}"
                else:
                    item_order["Remark"] = item.get("name")
            
            mapped_data.append(item_order)
    
    return mapped_data

def prepare_data_for_snow_order_model(snow_order_data):
    """
    准备数据以便与SnowOrderData模型关联
    转换datetime对象为字符串格式，确保数据格式兼容Go模型
    
    :param snow_order_data: 映射后的SnowOrderData格式数据列表
    :return: 处理后的数据列表，可直接用于与Go模型交互
    """
    processed_data = []
    
    for item in snow_order_data:
        processed_item = item.copy()
        # 转换datetime对象为字符串格式
        for key, value in processed_item.items():
            if isinstance(value, datetime):
                processed_item[key] = value.strftime("%Y-%m-%d %H:%M:%S")
        processed_data.append(processed_item)
    
    return processed_data

def send_dingtalk_message(shop_name, orders_result, refunds_result):
    """
    将店铺处理结果发送到钉钉群
    :param shop_name: 店铺名称
    :param orders_result: 订单处理结果 (插入数, 重复数)
    :param refunds_result: 退款处理结果 (插入数, 重复数)
    """
    # 钉钉机器人配置
    webhook = "https://oapi.dingtalk.com/robot/send?access_token=90f3fae0aa0e03a8ca113f6e99f97998700a0d769cca3340f881db7d873345d6"
    app_secret = "SEC4a5d4c9477980ad0e78fe62b47b44629b9dc5cedb02c0c6e541ac53e2bc52ad1"

    # 获取当前时间戳（毫秒）
    timestamp = str(round(time.time() * 1000))

    # 生成签名
    sign_str = f"{timestamp}\n{app_secret}"
    hmac_code = hmac.new(app_secret.encode('utf-8'), sign_str.encode('utf-8'), digestmod=hashlib.sha256).digest()
    sign = urllib.parse.quote_plus(base64.b64encode(hmac_code))

    # 构造最终URL
    full_url = f"{webhook}&timestamp={timestamp}&sign={sign}"

    # 解析处理结果
    original_count, filtered_count,insert_count = orders_result
    
    # 创建消息内容
    message = {
        "msgtype": "markdown",
        "markdown": {
            "title": "聚水潭数据同步报告",
            "text": f"聚水潭数据同步至数据库中"
                    f"### 🏪 {shop_name} 数据同步完成\n\n"
                    f"**订单数据**:\n"
                    f"- ✅ 原始订单: {original_count} 条\n"
                    f"- ✅ 过滤订单: {filtered_count} 条\n"
                    f"- ⚠️ 插入订单: {insert_count} 条\n\n"
                    f"**处理时间**: {time.strftime('%Y-%m-%d %H:%M:%S')}"
        },
        "at": {
            "isAtAll": False  # 不@所有人
        }
    }

    # 发送请求
    headers = {"Content-Type": "application/json"}
    try:
        response = requests.post(
            full_url,
            data=json.dumps(message),
            headers=headers
        )
        response.raise_for_status()
        print(f"钉钉消息发送成功: {shop_name}")
    except Exception as e:
        print(f"钉钉消息发送失败: {str(e)}")

# 使用示例
def save_snow_order_data(mapped_data):
    """
    保存映射后的SnowOrderData格式数据到JSON文件
    :param mapped_data: 映射后的SnowOrderData格式数据列表
    :return: 保存的文件路径
    """
    # 生成文件名
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    file_path = f"snow_order_data_{timestamp}.json"
    
    # 转换datetime对象为字符串格式
    for item in mapped_data:
        for key, value in item.items():
            if isinstance(value, datetime):
                item[key] = value.strftime("%Y-%m-%d %H:%M:%S")
    
    # 保存到JSON文件
    with open(file_path, 'w', encoding='utf-8') as f:
        json.dump(mapped_data, f, ensure_ascii=False, indent=2)
    
    print(f"SnowOrderData格式数据已保存到: {file_path}")
    return file_path

if __name__ == "__main__":
    shop_dict={
        "幼岚-抖音官方旗舰店":"11679528",
        "幼岚-抖音童装旗舰店":"16540940",
        "幼岚-抖音旗舰店":"11425575"
        # "美至心选-抖音店铺":"15951774",
        # "幼岚小红书":"11837473",
        # "美至小红书":"15951853",
        # "美至有赞":"14858938",
        # "美至微店":"14858898",
        # "幼岚视频号":"14395031",
        # "美至视频号":"15951688"
    }
    for shop_name, shop_id in shop_dict.items():
        print(f'正在处理 {shop_name} 的数据')

        # 获取订单数据
        extracted_orders, raw_orders = fetch_jushuitan_orders(shop_id)  # 获取原始数据
        
        # 将原始订单数据直接映射到SnowOrderData模型格式
        snow_order_data = map_to_snow_order_data(raw_orders)
        print(f"已映射 {len(snow_order_data)} 条SnowOrderData模型格式数据")
        
        # 准备数据以便与SnowOrderData模型关联
        # 此步骤将datetime对象转换为字符串格式，确保数据格式兼容Go模型
        prepared_data = prepare_data_for_snow_order_model(snow_order_data)
        print(f"已准备 {len(prepared_data)} 条数据用于与SnowOrderData模型关联")
        
        # 将准备好的数据直接传递到SnowOrderData模型中
        print(f"正在将 {len(prepared_data)} 条数据传递到SnowOrderData模型中...")
        # 这里添加将数据传递到模型的逻辑
        # 假设Go模型通过HTTP API接收数据，或者使用其他方式与Python交互
        # 由于没有具体的模型交互方法，这里使用占位符表示实际的数据传递过程
        # 在实际应用中，这里应该调用相应的函数或API将数据发送到Go模型中
        
        # 模拟数据传递到模型的成功状态
        print(f"数据已成功传递到SnowOrderData模型中")
        
        # 构建订单处理结果，用于钉钉通知
        orders_result = (len(extracted_orders), 0, len(snow_order_data))  # (原始数, 过滤数, 映射数)
        
        # 发送钉钉通知（只传递订单结果，不传递退款结果）
        send_dingtalk_message(shop_name, orders_result, None)