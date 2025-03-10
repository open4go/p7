package user

import (
	"context"
	"errors"
	"fmt"
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

// ChartItem 暂时够用了
type ChartItem struct {
	// x name can be date or something...
	Name string
	A    int
	B    int
	C    int
	D    int
	E    int
	F    int
	G    int
}

func Map2ChartItem(dataMap map[string]int, day string) ChartItem {
	// 创建 ChartItem 实例
	item := ChartItem{
		Name: day, // 示例日期
	}
	// 将 map 的值赋给 ChartItem
	for key, value := range dataMap {
		switch key {
		case "a":
			item.A = value
		case "b":
			item.B = value
		case "c":
			item.C = value
		case "d":
			item.D = value
		case "e":
			item.E = value
		case "f":
			item.F = value
		case "g":
			item.G = value
		default:
			fmt.Printf("无效的键: %s\n", key)
		}
	}
	return item
}

const (
	basePrefix = "data:chart"
)

// DataSourceType 定义数据源类型
type DataSourceType int

const (
	OrderCount      DataSourceType = iota // 订单
	RegisteredUsers                       // 注册人数
	LoginCount                            // 登录次数
	ReturnCount                           // 退货次数
	CommentCount                          // 评论次数
)

// String 方法用于将 DataSourceType 转换为字符串
func (dst DataSourceType) String() string {
	switch dst {
	case OrderCount:
		return "order"
	case RegisteredUsers:
		return "users"
	case LoginCount:
		return "login"
	case ReturnCount:
		return "return"
	case CommentCount:
		return "comment"
	default:
		return "unknown"
	}
}

// validateHead 验证 head 的值
func validateHead(head []string) error {
	validValues := map[string]struct{}{
		"a": {},
		"b": {},
		"c": {},
		"d": {},
		"e": {},
		"f": {},
		"g": {},
	}

	// 检查顺序和有效性
	expectedOrder := []string{"a", "b", "c", "d", "e", "f", "g"}
	seen := make(map[string]bool)
	for _, value := range head {
		if _, valid := validValues[value]; !valid {
			return fmt.Errorf("invalid value: %s", value)
		}
		if seen[value] {
			return errors.New("duplicate value found: " + value)
		}
		seen[value] = true

		// 检查顺序
		expectedIndex := -1
		for i, expected := range expectedOrder {
			if expected == value {
				expectedIndex = i
				break
			}
		}
		if expectedIndex == -1 || (len(seen) > 0 && expectedIndex < len(seen)-1) {
			return errors.New("values must appear in order and without gaps")
		}
	}

	// 检查是否超过 g
	if len(head) > len(expectedOrder) {
		return errors.New("head cannot have more than 7 values")
	}

	return nil
}

// NewChart
// 根据业务需求输入source
// 根据实际需要展示的数据维度，输入head
func NewChart(ctx context.Context, source DataSourceType, head []string) (*ChartData, error) {

	if err := validateHead(head); err != nil {
		return nil, err
	}

	return &ChartData{
		// 默认
		Expire:     7 * 24 * time.Hour,
		DataSource: source,
		Ctx:        ctx,
		// 数据类型，例如：订单数据，分为虚拟订单，线上订单，线下订单等
		Head: head,
	}, nil
}

type ChartData struct {
	DataSource DataSourceType `json:"source"` // 用户名
	Ctx        context.Context
	Expire     time.Duration
	Head       []string
}

func (d *ChartData) GetKey(t string) string {
	return fmt.Sprintf("%s:%s:%s", basePrefix, d.DataSource.String(), t)
}

// Push 存储时需要选择订单类型
// 订单类型t必须与head中的一个匹配
// vals 是默认的递增幅度，如果没有指定默认就是1，例如用户注册成功就增加1
// 对于订单销售额则可以指定金额，这里就不能输入小数点的数据了，需要取整
func (d *ChartData) Push(t string, vals ...int64) {
	today := time.Now().Format("2006-01-02")
	key := d.GetKey(t)

	// 设置默认值
	val := int64(1) // 默认值
	if len(vals) > 0 {
		val = vals[0] // 使用传入的值
	}

	err := d.push(d.Ctx, today, key, val)
	if err != nil {
		// TODO 暂时忽略错误
		return
	}
}

// push 用户数据存储
func (d *ChartData) push(ctx context.Context, today, key string, val int64) error {
	err := GetRedisCacheHandler(ctx).HIncrBy(ctx, key, today, val).Err()
	if err != nil {
		return err
	}

	// 设置过期时间为 7 天
	GetRedisCacheHandler(ctx).Expire(ctx, key, d.Expire)

	// 删除一个月前的数据
	oneMonthAgo := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	GetRedisCacheHandler(ctx).HDel(ctx, key, oneMonthAgo)

	// 其他用户注册逻辑
	return nil
}

// Stats 用户数据统计
func (d *ChartData) Stats(ctx context.Context, days int) ([]ChartItem, error) {
	today := time.Now()
	daysData := make([]ChartItem, 0)
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")

		// 每次重新初始化，避免数据污染
		t2v := make(map[string]int)
		for _, t := range d.Head {
			key := d.GetKey(t)
			fmt.Println("head key", key, date)
			count, err := GetRedisCacheHandler(ctx).HGet(ctx, key, date).Result()
			if err != nil {
				continue
			} else {
				countInt, _ := strconv.Atoi(count)
				t2v[t] = countInt
			}
		}

		item := Map2ChartItem(t2v, date)
		daysData = append(daysData, item)
	}
	return daysData, nil
}

func (d *ChartData) CurrentMonthly(ctx context.Context) ([]ChartItem, error) {
	daysData := make([]ChartItem, 0)

	// 遍历从本月的第一天到今天的每一天
	days, err := CurrentMonthlyDays(ctx)
	if err != nil {
		return daysData, err
	}
	for _, dateStr := range days {
		// 每次重新初始化，避免数据污染
		t2v := make(map[string]int)
		for _, t := range d.Head {
			key := d.GetKey(t)
			fmt.Println("head key", key, dateStr)
			count, err := GetRedisCacheHandler(ctx).HGet(ctx, key, dateStr).Result()
			if err != nil {
				continue
			}
			countInt, _ := strconv.Atoi(count)
			t2v[t] = countInt
		}

		item := Map2ChartItem(t2v, dateStr)
		daysData = append(daysData, item)
	}
	return daysData, nil
}

func (d *ChartData) CurrentMonthlySummary(ctx context.Context) (map[string]int, error) {
	// 遍历从本月的第一天到今天的每一天
	data, err := d.CurrentMonthly(ctx)
	if err != nil {
		return nil, err
	}

	// 将一个月的所有同一纬度的数据累加
	m := make(map[string]int)
	for _, i := range data {
		m["a"] = m["a"] + i.A
		m["b"] = m["b"] + i.B
		m["c"] = m["c"] + i.C
		m["d"] = m["d"] + i.D
		m["e"] = m["e"] + i.E
		m["f"] = m["f"] + i.F
		m["g"] = m["g"] + i.G
	}

	// 返回即可
	// 可以通过快速读取map值获得数据
	return m, nil
}

func CurrentMonthlyDays(ctx context.Context) ([]string, error) {
	today := time.Now()
	startOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	days := make([]string, 0)

	// 遍历从本月的第一天到今天的每一天
	for date := startOfMonth; date.Before(today) || date.Equal(today); date = date.AddDate(0, 0, 1) {
		dateStr := date.Format("2006-01-02")
		days = append(days, dateStr)
		fmt.Println("dateStr key", dateStr)
	}
	return days, nil
}
