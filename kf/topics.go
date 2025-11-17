package kf

// 常见topics 定义
const (
	// UserCreated 用户创建
	UserCreated = "user-created"

	// OrderCreated 订单创建
	OrderCreated = "order-created"
	// OrderCancel 订单取消
	OrderCancel = "order-cancel"
	// OrderPaying 订单支付中（未收到回调或者返回，这类订单需要主动查询订单）
	OrderPaying = "order-paying"
	// OrderPaySuccess 订单支付成功
	OrderPaySuccess = "order-pay-success"
	// OrderPayFailed 订单支付失败
	OrderPayFailed = "order-pay-failed"
	// OrderPrint 订单打印
	OrderPrint = "order-print"

	// FeedbackComment 反馈评价
	FeedbackComment = "feedback-comment"
	// RefundApply 退款申请
	RefundApply = "refund-apply"
)
