package user

import (
	"context"
	"strconv"
	"time"
)

type User struct {
	ID       string
	Username string
}

type DaysData struct {
	Date    string
	Counter int
}

const (
	dailyNewUser = "data:daily:new:users"
	onlineUser   = "data:online:users"
)

// Register 用户注册
func Register(ctx context.Context, username string) error {
	today := time.Now().Format("2006-01-02")
	err := GetRedisCacheHandler(ctx).HIncrBy(ctx, dailyNewUser, today, 1).Err()
	if err != nil {
		return err
	}

	// 设置过期时间为 7 天
	GetRedisCacheHandler(ctx).Expire(ctx, dailyNewUser, 7*24*time.Hour)

	// 删除一个月前的数据
	oneMonthAgo := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	GetRedisCacheHandler(ctx).HDel(ctx, dailyNewUser, oneMonthAgo)

	// 其他用户注册逻辑
	return nil
}

// GetStats 获取最近指定天数的用户统计
func GetStats(ctx context.Context, days int) (map[string]int, []DaysData, error) {
	today := time.Now()
	stats := make(map[string]int)
	daysData := make([]DaysData, 0)

	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		count, err := GetRedisCacheHandler(ctx).HGet(ctx, dailyNewUser, date).Result()
		if err != nil {
			stats[date] = 0
			daysData = append(daysData, DaysData{
				Date:    date,
				Counter: 0,
			})
		} else {
			countInt, _ := strconv.Atoi(count)
			stats[date] = countInt
			daysData = append(daysData, DaysData{
				Date:    date,
				Counter: countInt,
			})
		}
	}

	return stats, daysData, nil
}

// GetOnlineCount 获取当前在线人数
func GetOnlineCount(ctx context.Context) (int64, error) {
	return GetRedisCacheHandler(ctx).SCard(ctx, onlineUser).Result()
}

// Login 用户登录
func Login(ctx context.Context, userID string) error {
	return GetRedisCacheHandler(ctx).SAdd(ctx, onlineUser, userID).Err()
}

// Logout 用户登出
func Logout(ctx context.Context, userID string) error {
	return GetRedisCacheHandler(ctx).SRem(ctx, onlineUser, userID).Err()
}
