package user

import (
	"context"
	"encoding/json"
	"github.com/open4go/log"
	"github.com/open4go/req5rsp/cst"
	"github.com/open4go/req5rsp/rsp"
)

// PublishEvent 快速发布新的事件
func PublishEvent(ctx context.Context, d rsp.WSEvent) {
	payload, err := json.Marshal(d)
	if err != nil {
		log.Log(ctx).Error(err)
		return
	}
	err = GetRedisCacheHandler(ctx).Publish(ctx, cst.WSDataEvent, payload).Err()
	if err != nil {
		log.Log(ctx).Error(err)
		return
	}
}
