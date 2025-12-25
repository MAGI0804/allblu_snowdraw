import hashlib
import urllib.parse


def generate_sign_advanced(params, client_secret, exclude_keys=None):
    """
    增强版的签名生成方法

    Args:
        params (dict): 包含所有参数的字典
        client_secret (str): 专属的client_secret
        exclude_keys (list): 需要排除的键列表(如sign本身)

    Returns:
        str: 大写的MD5签名值
    """
    if exclude_keys is None:
        exclude_keys = ['sign']  # 默认排除sign参数

    # 过滤掉需要排除的参数
    filtered_params = {k: v for k, v in params.items()
                       if k not in exclude_keys and v is not None}

    # 步骤1：参数排序
    sorted_params = sorted(filtered_params.items(), key=lambda x: x[0])

    # 步骤2：字符串拼接
    base_string = ''.join([f"{key}{value}" for key, value in sorted_params])
    final_string = f"{client_secret}{base_string}{client_secret}"

    # 步骤3：生成sign值
    md5_hash = hashlib.md5(final_string.encode('utf-8'))
    sign = md5_hash.hexdigest().upper()

    return sign


def verify_sign(params, client_secret, received_sign):
    """
    验证签名是否正确

    Args:
        params (dict): 接收到的参数
        client_secret (str): 专属的client_secret
        received_sign (str): 接收到的签名

    Returns:
        bool: 签名是否正确
    """
    calculated_sign = generate_sign_advanced(params, client_secret)
    return calculated_sign == received_sign


# 使用示例
if __name__ == "__main__":
    # 测试数据
    test_params = {
        "type": "pdd.refund.list.increment.get",
        "timestamp": "1763098573",
        "client_id": "4b415953a5294085b1559afc0c453cb7",
        # "code": "471137caf97249bba0f0d0079b41846d3f173fce",
        "data_type":"json",
        "access_token":"3bab67a2267e469b8af8c650f3f01c1f0f68d26b",
        "page":"1",
        "page_size":"100",
        "after_sales_status":"10",
        "after_sales_type": "1",
        "start_updated_at":"1762963200",
        "end_updated_at":"1762965000"
        # "version": "1.0"
    }

    client_secret = "c584c4924f5ed15e393f1f16cb30993c12a655ad"

    # 生成签名
    sign = generate_sign_advanced(test_params, client_secret)
    print(f"签名: {sign}")

    # 验证签名
    is_valid = verify_sign(test_params, client_secret, sign)
    print(f"签名验证: {'通过' if is_valid else '失败'}")