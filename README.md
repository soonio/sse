# 消息推送模块

> 基于 SSE 实现分布式服务端消息推送服务

## 设计

  - 专注消息推送
    - 只关心消息服务这块，不关心业务逻辑
  - 用户认证
    - 需要注入一个用户认证逻辑
    - 识别用户ID，关联用户与sse链接
  - 消息订阅 
    - 主要关注个人模块的消息订阅
    - 不管群组
  - 消息推送
    - 基于redis发布订阅推送消息
  - 集群部署
    - 需要支持多台设备部署，协同工作


## 依赖redis

```bash
docker run --rm --name red -d -p 6379:6379 redis:alpine
```


## 功能模块

 - token认证
 - 消息分发

## 配置文件

## Proxy 代理配置

 - 注意 http版本要为1.1
 - 关闭缓冲区
 - 超时断开重连机制

```nginx configuration
upstream sse_group {
   server 172.28.95.169:9184;
   server 172.28.95.169:9185;
}

server {
   location ^~ /sse/ {
        rewrite ^/sse(.*) $1 break;
        proxy_pass http://sse_group;
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_read_timeout 43200s;
    }
}

```

## 进程守护

```unit file (systemd)
#/lib/systemd/system/sse.service

[Unit]
Description=sse service

[Service]
User=root
Type=simple
Restart=always
RestartSec=5s
ExecStart=/var/www/sse/bin -f config2.yml
ExecReload=/bin/kill -s HUP
WorkingDirectory=/var/www/sse
KillMode=process
Delegate=yes
StandardOutput=file:/var/log/sse.log
StandardError=file:/var/log/sse.log

[Install]
WantedBy=multi-user.target
```