package atten

import (
	"context"
	"errors"
	"fmt"
	"github.com/open4go/r3time"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"strings"
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
	now := r3time.LocTime()
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
	now := r3time.LocTime()
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

type AttendanceStatus struct {
	Bits         []int
	Counter      int64
	IsCheckToday bool // 当天是否已签到（新增字段）
}

// CheckStatusDetail 查询签到状态
// ym: 2025:04
func CheckStatusDetail(ctx context.Context, ym string, userID string) (AttendanceStatus, error) {

	a := AttendanceStatus{}

	key := fmt.Sprintf("attendance:%s:%s", userID, ym)
	// 查询本月签到状态
	status := make([]int, 31)
	for day := 1; day <= 31; day++ {
		bit, err := GetRedisCacheHandler(ctx).GetBit(ctx, key, int64(day-1)).Result()
		if err != nil {
			return a, errors.New("查询失败")
		}
		status[day-1] = int(bit)
	}

	a.Bits = status

	// 统计签到次数
	count, _ := GetRedisCacheHandler(ctx).BitCount(ctx, key, nil).Result()
	a.Counter = count

	// 5. 检查当天是否签到（新增）
	// 1. 验证日期格式
	parts := strings.Split(ym, ":")
	if len(parts) != 2 {
		return a, errors.New("日期格式错误，应为 year:month")
	}
	now := r3time.LocTime()
	currentYear, currentMonth, currentDay := now.Date()
	inputYear, _ := strconv.Atoi(parts[0])
	inputMonth, _ := strconv.Atoi(parts[1])

	// 只有当查询的是当前年月时才检查当天
	if inputYear == currentYear && inputMonth == int(currentMonth) {
		a.IsCheckToday = a.Bits[currentDay-1] == 1
	} else {
		a.IsCheckToday = false
	}
	// 查询状态
	return a, nil
}
