package user

import (
	"context"
	"fmt"
	"time"
)

func InitEventTypeValueReturnLatest(ctx context.Context, t int) int {
	key := fmt.Sprintf("cache:counter:%d", t)
	v, err := GetRedisCacheHandler(ctx).Get(ctx, key).Int()
	if err != nil {
		return 0
	}
	return v
}

func SetEventTypeValueReturnLatest(ctx context.Context, t int) int {
	key := fmt.Sprintf("cache:counter:%d", t)
	// a 申请取消；b：同意退款（待申请退款）
	GetRedisCacheHandler(ctx).IncrBy(ctx, key, 1)
	GetRedisCacheHandler(ctx).Expire(ctx, key, time.Hour*24*72)
	v, err := GetRedisCacheHandler(ctx).Get(ctx, key).Int()
	if err != nil {
		return 0
	}
	return v
}

// ViewEventTypeValue 当事件被查看后会清空
func ViewEventTypeValue(ctx context.Context, t int) {
	key := fmt.Sprintf("cache:counter:%d", t)
	// a 申请取消；b：同意退款（待申请退款）
	GetRedisCacheHandler(ctx).Del(ctx, key)
}
