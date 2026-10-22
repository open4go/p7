# p7 — 消息队列、签到、统计与排队组件

> 仓库：https://github.com/open4go/p7 ｜ 模块：`github.com/open4go/p7` ｜ Go 1.24.0 ｜ 最近提交：2026-08-18（最新 tag 在 v0.x，共 60 个 tag）

## 项目简介
业务中间组件合集：Kafka 生产/消费（`kf`）、RabbitMQ 收发（`mq`）、基于 Redis bitmap 的月度签到（`atten`）、按天的用户/事件统计图表数据和 WS 事件发布（`user`）、按商户估算等待时间的订单排队系统（`oqueue`）。

## 目录结构
```
p7/
├── kf/      producer.go(WriterManager: InitWriterManager/SendString/SendBytes/CloseAll) consumer.go(InitReader/GetReader/ConsumeLoop/CloseReader) topics.go(user-created、order-*、refund-apply…) demo.go
├── mq/      mq.go（Send/Receive/Process，amqp.url）
├── atten/   attendance.go（CheckIn/UndoCheckIn/CheckStatus/CheckStatusDetail，key attendance:{uid}:{y}:{m}）
├── user/    stat.go（NewChart/Push/Stats/CurrentMonthly…）cache.go（事件计数）publish.go（Redis Pub/Sub cst.WSDataEvent）
└── oqueue/  queue.go（QueueSystem：EnqueueOrder/GetOrderPosition/CompleteOrder/DequeueOrder/GetMerchantQueueStatusWithEstimate）
```

## 使用示例
```go
kf.InitWriterManager(ctx, viper.GetStringSlice("kafka.brokers"))
_ = kf.SendBytes(ctx, kf.OrderPaySuccess, payload)

_ = kf.InitReader(ctx, brokers, kf.OrderPrint, "printer-svc")
_ = kf.ConsumeLoop(ctx, kf.OrderPrint, func(b []byte) { ... })

if err := atten.CheckIn(ctx, userID); err != nil { ... }   // “已经签到过了”
qs := oqueue.NewQueueSystem(rdb)
_ = qs.EnqueueOrder(ctx, oqueue.OrderInfo{MerchantID: mid, OrderID: oid, NumOfItems: 3})
pos, wait, _, _ := qs.GetOrderPosition(ctx, oid)
```

## 依赖
`segmentio/kafka-go`、`rabbitmq/amqp091-go v1.10.0`、`go-redis/v9 v9.7.1`、`mongo-driver v1.15.0`、`open4go/{db v0.0.11, log v0.0.16, r3time v0.0.7, req5rsp v0.1.47}`

## 构建
`go build`、`go vet` 通过；没有测试；CI 最近一次失败。

## 与其他仓库的关系
依赖 `req5rsp`（rsp/cst）、`db`（Redis "cache"）、`r3time`；排队功能和 `util9s`、`r2id` 重复。
