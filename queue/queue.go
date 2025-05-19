package queuing

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

const (
	queueKeyFormat     = "queue:%s"
	statsKeyFormat     = "merchant_stats:%s"
	defaultExpiration  = 48 * time.Hour
	baseProcessTime    = 2 * time.Minute    // 每单基础处理时间
	itemProcessTimeKey = "avg_item_time"    // 平均每商品处理时间(毫秒)
	orderCountKey      = "processed_orders" // 已处理订单数
	defaultItemTime    = 1 * time.Minute    // 默认每商品处理时间
	minProcessedOrders = 5                  // 最小样本数才使用历史数据
)

// OrderInfo 增强版OrderInfo
type OrderInfo struct {
	MerchantID  string
	OrderID     string
	NumOfItems  int
	Timestamp   int64 // 入队时间戳
	EnqueueTime time.Time
}

// QueueSystem 增强版排队系统
type QueueSystem struct {
	client *redis.Client
}

// NewQueueSystem 创建队列系统
func NewQueueSystem(client *redis.Client) *QueueSystem {
	return &QueueSystem{client: client}
}

// getQueueKey 获取当天的队列key
func (q *QueueSystem) getQueueKey() string {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	date := time.Now().In(loc).Format("2006-01-02")
	return fmt.Sprintf(queueKeyFormat, date)
}

// getStatsKey 获取商家统计key
func (q *QueueSystem) getStatsKey(merchantID string) string {
	return fmt.Sprintf(statsKeyFormat, merchantID)
}

// EnqueueOrder 将订单加入队列
func (q *QueueSystem) EnqueueOrder(ctx context.Context, order OrderInfo) error {
	queueKey := q.getQueueKey()

	if err := q.client.Expire(ctx, queueKey, defaultExpiration).Err(); err != nil {
		return fmt.Errorf("failed to set expiration: %v", err)
	}

	if order.Timestamp == 0 {
		order.Timestamp = time.Now().UnixNano()
	}
	if order.EnqueueTime.IsZero() {
		order.EnqueueTime = time.Now()
	}

	value := fmt.Sprintf("%s:%s:%d:%d", order.MerchantID, order.OrderID, order.NumOfItems, order.EnqueueTime.Unix())

	// 修正后的ZAdd调用方式
	z := redis.Z{
		Score:  float64(order.Timestamp),
		Member: value,
	}

	// 两种调用方式，根据你的redis客户端版本选择一种
	// 方式1: 适用于较新版本
	if err := q.client.ZAdd(ctx, queueKey, z).Err(); err != nil {
		return fmt.Errorf("failed to enqueue order: %v", err)
	}

	return nil
}

// GetOrderPosition 获取订单在队列中的位置及预估等待时间
func (q *QueueSystem) GetOrderPosition(ctx context.Context, orderID string) (position int64, waitTime time.Duration, info *OrderInfo, err error) {
	queueKey := q.getQueueKey()

	orders, err := q.client.ZRangeWithScores(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to get queue: %v", err)
	}

	var found bool
	var precedingItems int
	var precedingOrders int

	for i, order := range orders {
		orderInfo, err := parseOrderInfo(order.Member.(string))
		if err != nil {
			continue
		}

		if orderInfo.OrderID == orderID {
			position = int64(i)
			info = orderInfo
			found = true
			break
		}

		// 计算前面订单的总商品数和订单数
		precedingItems += orderInfo.NumOfItems
		precedingOrders++
	}

	if !found {
		return 0, 0, nil, fmt.Errorf("order not found")
	}

	// 计算预估等待时间
	waitTime, err = q.calculateWaitTime(ctx, info.MerchantID, precedingItems, precedingOrders)
	if err != nil {
		return position, 0, info, fmt.Errorf("failed to calculate wait time: %v", err)
	}

	return position, waitTime, info, nil
}

// calculateWaitTime 计算预估等待时间
func (q *QueueSystem) calculateWaitTime(ctx context.Context, merchantID string, precedingItems, precedingOrders int) (time.Duration, error) {
	statsKey := q.getStatsKey(merchantID)

	// 获取商家历史统计数据
	avgItemTime, processedOrders, err := q.getMerchantStats(ctx, statsKey)
	if err != nil {
		return 0, err
	}

	// 如果历史数据不足，使用默认值
	if processedOrders < minProcessedOrders {
		avgItemTime = defaultItemTime
	}

	// 计算预估时间 = (前面商品数 × 平均每商品时间) + (前面订单数 × 基础处理时间)
	totalWait := time.Duration(precedingItems)*avgItemTime + time.Duration(precedingOrders)*baseProcessTime

	return totalWait, nil
}

// getMerchantStats 获取商家历史统计数据
func (q *QueueSystem) getMerchantStats(ctx context.Context, statsKey string) (avgItemTime time.Duration, processedOrders int64, err error) {
	results, err := q.client.HMGet(ctx, statsKey, itemProcessTimeKey, orderCountKey).Result()
	if err != nil {
		return 0, 0, err
	}

	// 解析平均每商品处理时间(毫秒)
	if results[0] != nil {
		itemTimeMs, _ := strconv.ParseInt(results[0].(string), 10, 64)
		avgItemTime = time.Duration(itemTimeMs) * time.Millisecond
	}

	// 解析已处理订单数
	if results[1] != nil {
		processedOrders, _ = strconv.ParseInt(results[1].(string), 10, 64)
	}

	return avgItemTime, processedOrders, nil
}

// CompleteOrder 订单完成处理，更新统计数据
func (q *QueueSystem) CompleteOrder(ctx context.Context, orderID string) error {
	// 先获取订单信息
	_, _, info, err := q.GetOrderPosition(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order info: %v", err)
	}

	// 从队列中移除
	if err := q.DequeueOrder(ctx, orderID); err != nil {
		return fmt.Errorf("failed to dequeue order: %v", err)
	}

	// 计算处理时间
	processDuration := time.Since(info.EnqueueTime)

	// 更新商家统计数据
	return q.updateMerchantStats(ctx, info.MerchantID, info.NumOfItems, processDuration)
}

// updateMerchantStats 更新商家统计数据
func (q *QueueSystem) updateMerchantStats(ctx context.Context, merchantID string, numOfItems int, processDuration time.Duration) error {
	statsKey := q.getStatsKey(merchantID)

	// 获取当前统计数据
	avgItemTime, processedOrders, err := q.getMerchantStats(ctx, statsKey)
	if err != nil {
		return err
	}

	// 计算每商品平均处理时间(毫秒)
	itemTime := processDuration / time.Duration(numOfItems)

	// 更新平均值(加权平均)
	var newAvgItemTime time.Duration
	if processedOrders == 0 {
		newAvgItemTime = itemTime
	} else {
		// 新平均值 = (旧平均值 × 旧数量 + 新值) / (旧数量 + 1)
		total := avgItemTime*time.Duration(processedOrders) + itemTime
		newAvgItemTime = total / time.Duration(processedOrders+1)
	}

	// 保存更新后的数据
	_, err = q.client.HSet(ctx, statsKey,
		itemProcessTimeKey, fmt.Sprintf("%d", newAvgItemTime.Milliseconds()),
		orderCountKey, fmt.Sprintf("%d", processedOrders+1),
	).Result()

	// 设置过期时间
	q.client.Expire(ctx, statsKey, 30*24*time.Hour) // 保留30天统计数据

	return err
}

// parseOrderInfo 解析订单信息(增强版)
func parseOrderInfo(s string) (*OrderInfo, error) {
	var merchantID, orderID string
	var numOfItems int
	var enqueueTimeUnix int64

	n, err := fmt.Sscanf(s, "%s:%s:%d:%d", &merchantID, &orderID, &numOfItems, &enqueueTimeUnix)
	if err != nil || n != 4 {
		return nil, fmt.Errorf("invalid order info format")
	}

	return &OrderInfo{
		MerchantID:  merchantID,
		OrderID:     orderID,
		NumOfItems:  numOfItems,
		EnqueueTime: time.Unix(enqueueTimeUnix, 0),
	}, nil
}

// DequeueOrder 保留原有方法(GetMerchantQueueStatus, DequeueOrder等)...
// DequeueOrder 将订单从队列中移除
func (q *QueueSystem) DequeueOrder(ctx context.Context, orderID string) error {
	queueKey := q.getQueueKey()

	// 先找到订单
	orders, err := q.client.ZRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get queue: %v", err)
	}

	var target string
	for _, order := range orders {
		info, err := parseOrderInfo(order)
		if err != nil {
			continue
		}

		if info.OrderID == orderID {
			target = order
			break
		}
	}

	if target == "" {
		return fmt.Errorf("order not found")
	}

	// 从有序集合中移除
	if err := q.client.ZRem(ctx, queueKey, target).Err(); err != nil {
		return fmt.Errorf("failed to dequeue order: %v", err)
	}

	return nil
}

// GetMerchantQueueStatus 获取商家的队列状态
func (q *QueueSystem) GetMerchantQueueStatus(ctx context.Context, merchantID string) (int, int, error) {
	queueKey := q.getQueueKey()

	// 获取所有订单
	orders, err := q.client.ZRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get queue: %v", err)
	}

	var count int
	var totalItems int

	for _, order := range orders {
		info, err := parseOrderInfo(order)
		if err != nil {
			continue
		}

		if info.MerchantID == merchantID {
			count++
			totalItems += info.NumOfItems
		}
	}

	return count, totalItems, nil
}

// GetMerchantQueueStatusWithEstimate 获取商家队列状态及新订单预估等待时间
func (q *QueueSystem) GetMerchantQueueStatusWithEstimate(ctx context.Context, merchantID string, newOrderItems int) (orderCount int, totalItems int, estimatedWait time.Duration, err error) {
	queueKey := q.getQueueKey()

	// 获取当前队列中该商家的所有订单
	orders, err := q.client.ZRange(ctx, queueKey, 0, -1).Result()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get queue: %v", err)
	}

	// 统计当前队列状态
	for _, order := range orders {
		info, err := parseOrderInfo(order)
		if err != nil {
			continue
		}

		if info.MerchantID == merchantID {
			orderCount++
			totalItems += info.NumOfItems
		}
	}

	// 计算预估等待时间
	estimatedWait, err = q.estimateNewOrderWaitTime(ctx, merchantID, totalItems, orderCount, newOrderItems)
	if err != nil {
		return orderCount, totalItems, 0, fmt.Errorf("failed to estimate wait time: %v", err)
	}

	return orderCount, totalItems, estimatedWait, nil
}

// estimateNewOrderWaitTime 预估新订单等待时间
func (q *QueueSystem) estimateNewOrderWaitTime(ctx context.Context, merchantID string, currentItems, currentOrders, newOrderItems int) (time.Duration, error) {
	statsKey := q.getStatsKey(merchantID)

	// 获取商家历史统计数据
	avgItemTime, processedOrders, err := q.getMerchantStats(ctx, statsKey)
	if err != nil {
		return 0, err
	}

	// 如果历史数据不足，使用默认值
	if processedOrders < minProcessedOrders {
		avgItemTime = defaultItemTime
	}

	// 预估时间计算逻辑:
	// 1. 当前队列中所有订单的处理时间
	currentWait := time.Duration(currentItems)*avgItemTime + time.Duration(currentOrders)*baseProcessTime

	// 2. 新订单本身的处理时间(按历史平均效率计算)
	newOrderProcessTime := time.Duration(newOrderItems)*avgItemTime + baseProcessTime

	// 3. 总预估时间 = 当前队列处理时间 + 新订单处理时间
	// (这里假设新订单会排在最后)
	totalWait := currentWait + newOrderProcessTime

	// 增加20%缓冲时间(可根据实际情况调整)
	totalWait = time.Duration(float64(totalWait) * 1.2)

	return totalWait, nil
}
