# SSE 

> 基于 sse 实现服务端消息推送

## 启动服务

```bash
go run main.go
```

## 消息模拟

### 接收消息

```bash
curl --location 'http://127.0.0.1:8080/sse?uid=111'
```
### 发送消息

```bash
curl --location 'http://127.0.0.1:8080/tips?uid=111&msg=%E6%88%91%E6%AC%A1%E5%A5%A5%E5%8E%89%E5%AE%B3%E5%95%A6'
```
