import requests,time,hashlib



def md5_encrypt(payment_str):   #进行MD5加密
    md5_hash = hashlib.md5()
    md5_hash.update(payment_str.encode('utf-8'))
    return md5_hash.hexdigest().lower()

def get_token():
    app_key = "e50a8f2e66c845c188a04f34ebf4a663"
    timestamp = int(time.time())
    charset = "uft-8"
    app_select = 'b7a7e5df75ed4ae38c42db4fbe060fb8'
    grant_type = "authorization_code"
    code = "4xFIOC"
    converted_str = f'{app_select}app_key{app_key}charset{charset}code{code}grant_type{grant_type}timestamp{timestamp}'
    sign = md5_encrypt(converted_str)
    url = "https://openapi.jushuitan.com/openWeb/auth/getInitToken"
    headers = {
        "Content-Type": "application/x-www-form-urlencoded"
    }
    data = {
        "app_key": app_key,
        "grant_type": grant_type,
        "timestamp": timestamp,
        "code": code,
        "charset": charset,
        "sign": sign
    }
    # print(data)
    try:
        response = requests.post(url, headers=headers, data=data)
        response.raise_for_status()  # 抛出HTTP错误
        return response.json()["data"]["access_token"]
    except requests.exceptions.RequestException as e:
        print(f"请求发送失败: {str(e)}")
        return None







if __name__ == '__main__':
    json_reponse = get_token()
    print(json_reponse)

