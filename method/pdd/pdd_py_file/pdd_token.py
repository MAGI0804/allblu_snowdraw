import hashlib
import time
import requests
from pdd_sign import *

import requests
import time
import hashlib


def get_pdd_access_token(client_id, client_secret, code, timestamp=None):
    """
    获取拼多多access_token

    Args:
        client_id (str): 应用ID
        client_secret (str): 应用密钥
        code (str): 授权码，不同店铺会不一样
        timestamp (str): 时间戳，如果不提供则使用当前时间

    Returns:
        dict: 包含access_token等信息的响应数据
    """

    # 拼多多API地址
    url = "https://gw-api.pinduoduo.com/api/router"

    # 准备请求参数
    if timestamp is None:
        timestamp = str(int(time.time()))

    params = {
        "type": "pdd.pop.auth.token.create",
        "client_id": client_id,
        "timestamp": timestamp,
        "code": code
    }

    # 生成签名
    sign = generate_sign_advanced(params, client_secret)
    params["sign"] = sign

    try:
        # 发送POST请求
        response = requests.post(url, data=params)
        response.raise_for_status()

        # 解析响应
        result = response.json()

        # 检查是否有错误
        if "error_response" in result:
            error_msg = result["error_response"]
            raise Exception(
                f"API错误: {error_msg.get('error_msg', '未知错误')} (代码: {error_msg.get('error_code', '未知')})")

        # 返回token信息
        if "pop_auth_token_create_response" in result:
            token_data = result["pop_auth_token_create_response"]

            # 返回完整的token信息字典，包含所有有用字段
            return {
                'success': True,
                'access_token': token_data.get('access_token'),
                'refresh_token': token_data.get('refresh_token'),
                'expires_at': token_data.get('expires_at'),
                'owner_name': token_data.get('owner_name'),
                'owner_id': token_data.get('owner_id'),
                'scope': token_data.get('scope'),
                'token_type': token_data.get('token_type'),
                'raw_response': token_data  # 包含原始响应数据
            }
        else:
            raise Exception("响应格式不正确")

    except requests.exceptions.RequestException as e:
        return {
            'success': False,
            'error': f"网络请求失败: {str(e)}"
        }
    except ValueError as e:
        return {
            'success': False,
            'error': f"响应解析失败: {str(e)}"
        }
    except Exception as e:
        return {
            'success': False,
            'error': str(e)
        }


# 使用示例
def get_and_save_pdd_token(client_id, client_secret, code):
    """
    获取并保存拼多多token的完整流程

    Args:
        client_id (str): 应用ID
        client_secret (str): 应用密钥
        code (str): 授权码

    Returns:
        dict: 包含token信息和处理结果
    """
    result = get_pdd_access_token(client_id, client_secret, code)

    if result['success']:
        print("获取access_token成功:")
        print(f"Access Token: {result.get('access_token')}")
        print(f"Refresh Token: {result.get('refresh_token')}")
        print(f"过期时间: {result.get('expires_at')}")
        print(f"店铺名称: {result.get('owner_name')}")
        print(f"店铺ID: {result.get('owner_id')}")

        # 这里可以添加保存token的逻辑，比如保存到数据库或文件
        # save_token_to_db(result)

    else:
        print(f"获取access_token失败: {result.get('error')}")

    return result


# 直接调用示例
if __name__ == "__main__":
    # 配置参数 - 请替换为实际的值
    CLIENT_ID = "4b415953a5294085b1559afc0c453cb7"
    CLIENT_SECRET = "c584c4924f5ed15e393f1f16cb30993c12a655ad"
    CODE = "9eb8aa907d0c4c90b14b90e87390d1fa664cdf8b"

    # 获取token
    token_result = get_and_save_pdd_token(CLIENT_ID, CLIENT_SECRET, CODE)

    # 后续可以直接使用返回的token
    if token_result['success']:
        access_token = token_result['access_token']
        refresh_token = token_result['refresh_token']
        # 使用access_token调用其他API...