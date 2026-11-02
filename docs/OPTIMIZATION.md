# p7 优化方案

| # | 优先级 | 问题 | 位置 | 建议 | 工作量 |
|---|---|---|---|---|---|
| 1 | 高 | `mq.Send`/`Receive` 出错时 `log.Panicf`（`failOnError`），**进程会崩溃**；`Send` 每发一条消息都要新建一次 AMQP 连接和 channel；返回值恒为 nil | `mq/mq.go:13,25,72` | 返回 error；用长连接 + channel 池，断线自动重连；队列设为 durable | M |
| 2 | 高 | govulncheck 在 p7 中发现 15 个问题，其中 amqp091-go v1.10.0 有 9 个（GO-2026-6700/6503/6500/6498/...，涉及 DoS、OOM、数据截断） | `go.mod` | 升级 `rabbitmq/amqp091-go` 到 v1.14.0+；如果 RabbitMQ 已经被 Kafka 取代，就删除整个 `mq` 包 | S |
| 3 | 中 | 签到不是原子操作：先 `GetBit` 再 `SetBit`，并发请求可能都判断为未签到，导致奖励发两次 | `atten/attendance.go:31-41` | 直接 `SETBIT`，它会返回旧值，旧值为 1 就表示已经签到过 | S |
| 4 | 中 | 排队 key 的过期时间设在 ZAdd **之前**：当天第一单时 key 还不存在，Expire 不生效，这个 key 会永久保留 | `oqueue/queue.go:59` | 先 ZAdd 再 Expire（pipeline 或 TxPipeline） | S |
| 5 | 中 | 排队的 score 是 `float64(UnixNano)`（约 1.7e18，超过 2^53），精度丢失，同一微秒级时间内的顺序不稳定 | `oqueue/queue.go` EnqueueOrder | 改用毫秒或 Redis INCR 序号作为 score | S |
| 6 | 中 | 所有商户共用一个按天的队列 key（`queue:{date}`），计算位置时要扫描全部商户的数据 | `oqueue/queue.go:14,47` | key 改成 `queue:{merchant}:{date}` | M |
| 7 | 中 | `ConsumeLoop` 用 `ReadMessage` 自动提交 offset，handler 处理失败或 panic 时消息会丢失；读错误时没有退避，会高速打日志 | `kf/consumer.go` | 改为 `FetchMessage` + 处理成功后 `CommitMessages`；recover；指数退避 | M |
| 8 | 低 | 事件计数 `cache:counter:{t}` 没有按租户区分，而且 IncrBy + Expire + Get 分三步执行 | `user/cache.go` | key 加上商户维度；直接用 IncrBy 的返回值 | S |
| 9 | 低 | `kf/demo.go`、`mq.Process` 这类示例代码放在库包里 | — | 移到 example_test | S |
| 10 | 中 | 没有测试 | — | 用 miniredis 测签到和排队，用 kafka testcontainer 测生产消费 | M |
| 11 | 低 | 旧版 log v0.0.16 带来的 docker 依赖 | `go.mod` | 升级 log | S |
