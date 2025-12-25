import hashlib
from datetime import datetime

def get_formatted_time():
    # 获取当前时间
    current_time = datetime.now()
    # 格式化时间
    formatted_time_str = current_time.strftime("%Y-%m-%d %H:%M:%S")
    return formatted_time_str

def generate_md5_signature(params: dict) -> str:
    """
    生成MD5加密签名字符串
    步骤：
    1. 将字典按键名升序排序
    2. 拼接所有参数名和参数值
    3. 使用MD5加密
    4. 转换为大写十六进制字符串

    :param params: 待签名的参数字典
    :return: 大写的MD5十六进制签名
    """
    # 1. 按键名字母顺序排序
    app_secret='a9f5cd5174b0007500ecd99b3bbe6daf'
    sorted_params = sorted(params.items(), key=lambda x: x[0])

    # 2. 拼接键值对
    concatenated = ''.join(f"{key}{value}" for key, value in sorted_params)
    final_string=f'{app_secret}{concatenated}{app_secret}'

    # 3. 使用MD5加密
    m = hashlib.md5()
    m.update(final_string.encode('utf-8'))

    # 4. 转为大写十六进制
    return m.hexdigest().upper()


# 示例用法
if __name__ == "__main__":
    #获取当前时间
    current_time=datetime.now()
    #格式化时间
    format_time=current_time.strftime("%Y-%m-%d %H:%M:%S")
    # print(format_time)
    params = {
        "method":"taobao.logistics.orders.detail.get",
        # "method": "taobao.trades.sold.get",
        "app_key":34101613,
        "session":"6101607ec53f58e15fdbc10cc7baa1d2c3c4e292fce19183948263976",
        "timestamp":"2025-09-29 15:25:00",
        "v":2.0,
        "sign_method":"md5",
        # "cids":"50013189,121364004",
        "fields":"receiver_location,tid",
        "format":"json",
        "tid":"104308008934987"
        # "page_size":100,
        # "use_has_next":"true",
        # "fields":"orders,pay_time,receiver_name,receiver_state,receiver_city,receiver_district,receiver_town,receiver_address,receiver_mobile,service_orders,cid"
    }

    signature = generate_md5_signature(params)
    print(f"生成的签名: {signature}")