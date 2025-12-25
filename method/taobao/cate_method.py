import requests
import time
from collections import defaultdict
from taobao_order import *

def get_category_info(cid_list):
    """根据CID列表获取类目信息"""
    if not cid_list:
        return {}

    api_url = "https://gw.api.taobao.com/router/rest"
    app_key = "34101613"
    session = "6101607ec53f58e15fdbc10cc7baa1d2c3c4e292fce19183948263976"

    # 准备请求参数
    params = {
        "method": "taobao.itemcats.get",
        "app_key": app_key,
        "session": session,
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S", time.localtime()),
        "v": "2.0",
        "sign_method": "md5",
        "cids": ",".join(map(str, cid_list)),
        "format": "json",
        "fields": "cid,name,status"
    }

    # 生成签名
    params["sign"] = generate_md5_signature(params)

    try:
        response = requests.post(api_url, data=params, timeout=10)
        response.raise_for_status()
        result = response.json()

        # 解析返回数据
        category_map = {}
        response_data = result.get("itemcats_get_response", {})
        cats = response_data.get("item_cats", {}).get("item_cat", [])

        for cat in cats:
            if cat.get("status") == "normal":
                category_map[cat["cid"]] = cat["name"]

        return category_map

    except Exception as e:
        print(f"获取类目信息失败: {str(e)}")
        return {}


def replace_and_remove_cid(extracted_data):
    """将extracted_data中的CID替换为类目名称并移除CID字段"""
    if not extracted_data:
        return extracted_data

    # 1. 收集所有需要查询的CID并进行去重
    all_cids = set()
    for item in extracted_data:
        if "cid" in item:
            all_cids.add(item["cid"])

    # 2. 将去重后的CID列表转换为列表形式
    unique_cids = list(all_cids)
    print(f"共发现 {len(unique_cids)} 个唯一CID需要查询")

    # 3. 批量获取类目信息
    category_map = get_category_info(unique_cids)

    # 4. 创建默认字典处理缺失的CID
    default_map = defaultdict(lambda: "未知类目")
    default_map.update(category_map)

    # 5. 替换CID为类目名称并移除CID字段
    for item in extracted_data:
        if "cid" in item:
            cid = item["cid"]
            # 添加name字段
            item["name"] = default_map[cid]
            # 移除cid字段
            del item["cid"]
        # 如果条目中没有cid，添加默认的name字段
        elif "name" not in item:
            item["name"] = "未知类目"

    return extracted_data

if __name__ == "__main__":
    result = get_taobao_orders(
        fields=["tid", "orders", "pay_time", "oaid"]  # 字段列表
    )
    extracted_data = extract_order_data(result)
    # 执行替换和移除
    updated_data = replace_and_remove_cid(extracted_data)

    print("\n替换后数据:")
    for item in updated_data:
        print(item)