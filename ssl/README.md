# SSL证书放置说明

当前SSL证书文件已放置在此目录下，文件名称已重命名为与域名一致：

1. 证书文件：`shop_test.youlankids.com.pem`
2. 私钥文件：`shop_test.youlankids.com.key`

注意：nginx.conf中的SSL配置已更新为使用这些文件名。

这些文件将通过Docker Compose挂载到Nginx容器的`/app/ssl/`目录下，供HTTPS配置使用。

## 注意事项
1. 请确保证书文件具有正确的权限设置
2. 请勿将私钥文件泄露给未授权人员
3. 证书过期后请及时更新