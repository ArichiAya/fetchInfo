// Package model 定义跨服务共享的行情数据结构。
// 价格在 proto 中为 float(32位),Go 内部统一用 float64,在编解码边界转换。
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/ArichiAya/fetch-info/common/msg"

	"google.golang.org/protobuf/proto"

	"github.com/ArichiAya/fetch-info/market/internal/indicator"
)

type Tick2 struct {
	TickRow   *msg.Tick
	Delta_V   int64
	VWAP      float64
	P_Book    float64
	Slope_Bid float64
	Slope_Ask float64
	// Ret       float64 需要上一个tick的数据进行计算,查询时再计算
	Spread float64
}

// 生成并返回*Tick2.在tick的基础上计算指标
func InitTick2(msgDataBytes []byte) (*Tick2, error) {
	tickMsg := &msg.Tick{}

	if err := proto.Unmarshal(msgDataBytes, tickMsg); err != nil {
		return nil, fmt.Errorf("Unmarshal error: %w", err)
	}
	t2 := &Tick2{}

	t2.TickRow = tickMsg
	quoteslength := len(tickMsg.Quotes)
	iAVG := indicator.CaliAVG(tickMsg.Quotes)

	t2.Delta_V = indicator.CalDeltaV(tickMsg.Price, tickMsg.Quotes[0].AskP, tickMsg.Quotes[0].BidP, tickMsg.LastVolume)

	t2.VWAP = indicator.CalVWAP(tickMsg.CumAmount, float64(tickMsg.CumVolume))

	t2.P_Book = indicator.CalPBook(tickMsg.Quotes)

	t2.Slope_Bid = indicator.CalSlopeBid(tickMsg.Quotes, iAVG, indicator.CalbidvAVG(tickMsg.Quotes, float64(quoteslength)))

	t2.Slope_Ask = indicator.CalSlopeAsk(tickMsg.Quotes, iAVG, indicator.CalaskvAVG(tickMsg.Quotes, float64(quoteslength)))

	t2.Spread = indicator.CalSpread(tickMsg.Quotes[0].AskP, tickMsg.Quotes[0].BidP)

	return t2, nil
}

type Tick2Option func(*msg.Tick2)

func WithTickRow(tickrow *msg.Tick) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.TickRow = tickrow
	}
}

func WithDelta_V(delta_v int64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.DeltaV = delta_v
	}
}

func WithVWAP(vwap float64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.Vwap = vwap
	}
}

func WithP_Book(p_book float64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.PBook = p_book
	}
}

func WithSlope_Bid(slope_bid float64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.SlopeBid = slope_bid
	}
}

func WithSlope_Ask(slope_ask float64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.SlopeAsk = slope_ask
	}
}

func WithSpread(spread float64) Tick2Option {
	return func(t2 *msg.Tick2) {
		t2.Spread = spread
	}
}

func NewTick2(ops ...Tick2Option) *msg.Tick2 {
	t2 := &msg.Tick2{}

	for _, option := range ops {
		option(t2)
	}

	return t2
}

func (t2 *Tick2) Value() (driver.Value, error) {
	return json.Marshal(t2)
}
