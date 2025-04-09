package attendance

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Attendance struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    string             `bson:"userId"`
	Date      time.Time          `bson:"date"`
	CreatedAt time.Time          `bson:"createdAt"`
}

// CheckIn 签到
func CheckIn(ctx context.Context, userId string) error {
	now := time.Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()

	key := fmt.Sprintf("attendance:%s:%d:%d", userId, year, month)

	// 检查是否已经签到
	checkedIn, err := GetRedisCacheHandler(ctx).GetBit(ctx, key, int64(day-1)).Result()
	if err != nil {
		return errors.New("签到失败")
	}

	if checkedIn == 1 {
		return errors.New("已经签到过了")
	}

	// 签到
	_, err = GetRedisCacheHandler(ctx).SetBit(ctx, key, int64(day-1), 1).Result()
	if err != nil {
		return errors.New("签到失败")
	}

	// 设置键的过期时间为半年
	expiration := 183 * 24 * time.Hour // 半年
	_, err = GetRedisCacheHandler(ctx).Expire(ctx, key, expiration).Result()
	if err != nil {
		return errors.New("设置过期时间失败")
	}
	
	// 签到成功
	return nil
}

// CheckStatus 查询签到状态
func CheckStatus(ctx context.Context, userID string) ([]int, error) {
	now := time.Now()
	year := now.Year()
	month := now.Month()

	key := fmt.Sprintf("attendance:%s:%d:%d", userID, year, month)

	// 查询本月签到状态
	status := make([]int, 31)
	for day := 1; day <= 31; day++ {
		bit, err := GetRedisCacheHandler(ctx).GetBit(ctx, key, int64(day-1)).Result()
		if err != nil {
			return nil, errors.New("查询失败")
		}
		status[day-1] = int(bit)
	}

	// 查询状态
	return status, nil
}
