@echo off

REM 启用命令扩展
setlocal enabledelayedexpansion

REM 设置本地环境变量
set "DB_HOST=127.0.0.1"
set "DB_PORT=3306"
set "DB_NAME=wechat_member"
set "DB_USER=root"
set "DB_PASSWORD="
set "SECRET_KEY=django-insecure-m89*e^)$=h*0a#6^!$=e+r@w5%985^!j^!f8y2mt%@#qw%o=i#^!f*6"

REM 显示当前设置的环境变量
echo 当前设置的环境变量
echo DB_HOST=%DB_HOST%
echo DB_PORT=%DB_PORT%
echo DB_NAME=%DB_NAME%
echo DB_USER=%DB_USER%

echo 正在启动应用程序...

REM 运行应用程序并捕获输出
.\youlan_kids_go > app.log 2>&1

REM 等待应用程序运行一段时间
ping -n 3 127.0.0.1 > nul

REM 检查应用程序是否在运行
 echo 检查应用程序进程:
tasklist | findstr "youlan_kids_go"

REM 检查8080端口是否被占用
 echo 检查8080端口状态:
netstat -ano | findstr "8080"

REM 显示应用程序输出日志的前50行
echo 
 echo 应用程序输出日志:
more +1 app.log

pause