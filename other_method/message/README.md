# 发送短信完整工程示例

该项目为SendSms的完整工程示例。

该示例**无法在线调试**，如需调试可下载到本地后替换 [AK](https://usercenter.console.aliyun.com/#/manage/ak) 以及参数后进行调试。

## 运行条件

- 下载并解压需要语言的代码;


- 在阿里云帐户中获取您的 [凭证](https://usercenter.console.aliyun.com/#/manage/ak) 并通过它替换下载后代码中的 ACCESS_KEY_ID 以及 ACCESS_KEY_SECRET;

- 执行对应语言的构建及运行语句

## 执行步骤

下载的代码包，在根据自己需要更改代码中的参数和 AK 以后，可以在**解压代码所在目录下**按如下的步骤执行：

- *Go 环境版本必须不低于 1.10.x*
- *安装 SDK 核心库 OpenAPI*
```sh
go get github.com/alibabacloud-go/darabonba-openapi/v2/client
```
- *执行命令*
```sh
GOPROXY=https://goproxy.cn,direct go run ./main
```
## 使用的 API

-  SendSms：向指定的手机号码发送短信。 更多信息可参考：[文档](https://next.api.aliyun.com/document/Dysmsapi/2017-05-25/SendSms)

## API 返回示例

*实际输出结构可能稍有不同，属于正常返回；下列输出值仅作为参考，以实际调用为准*


- JSON 格式 
```js
{
  "Code": "OK",
  "Message": "OK",
  "BizId": "9006197469364984****",
  "RequestId": "F655A8D5-B967-440B-8683-DAD6FF8DE990"
}
```

