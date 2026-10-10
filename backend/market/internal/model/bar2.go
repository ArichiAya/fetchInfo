// Package model 定义跨服务共享的行情数据结构。
// 价格在 proto 中为 float(32位),Go 内部统一用 float64,在编解码边界转换。
package model

import (
	"fmt"

	"github.com/ArichiAya/fetch-info/common/msg"
	"github.com/ArichiAya/fetch-info/market/internal/indicator"
	"google.golang.org/protobuf/proto"
)

type Bar2 struct {
	BarRow *msg.Bar
	// TR float64 需要上一个bar的数据进行计算,查询时再计算
	BodyRatio float64
	// DeltaClose float64 需要上一个bar的数据进行计算,查询时再计算
}

func NewBar2(msgDataBytes []byte) (*Bar2, error) {
	barMsg := &msg.Bar{}

	if err := proto.Unmarshal(msgDataBytes, barMsg); err != nil {
		return nil, fmt.Errorf("Unmarshal error: %w", err)
	}

	b2 := &Bar2{}
	b2.BarRow = barMsg

	b2.BodyRatio = indicator.CalBodyRatio(barMsg.Open, barMsg.Close, barMsg.High, barMsg.Low)

	return b2, nil
}
