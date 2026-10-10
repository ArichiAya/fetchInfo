// Package indicator 实现降采样:消费 NATS 上的原始 tick 与  bar,
// 计算技术指标,
// 产出统一发布到 Kafka(market.tick / market.bar)。

package indicator

import (
	"math"

	"github.com/ArichiAya/fetch-info/common/msg"

	"github.com/ArichiAya/fetch-info/common/utils"
)

//---------------- Tick相关指标计算函数 ----------------//

// CalDeltaV 计算deltaV,deltaV用于后续计算CIS(累积主动买卖量差)
func CalDeltaV(price float64, askp0 float64, bidp0 float64, lastVolume int64) (deltav int64) {
	if price >= askp0 && askp0 > 0 {
		return lastVolume
	} else if price <= bidp0 && bidp0 > 0 {
		return -lastVolume
	} else {
		return 0
	}
}

// CalVWAP 计算VWAP(日内成交均价),在Tick里微观为瞬时 VWAP
func CalVWAP(cumAmount float64, cumVolume float64) (vwap float64) {
	return utils.SafeRatio(cumAmount, cumVolume, 0)
}

// CalPBook 计算五档订单簿重心(盘口加权价)
func CalPBook(quotes []*msg.Quote) (p_book float64) {
	var pbook1 float64
	var pbook2 float64
	for _, quote := range quotes {
		pbook1 += float64(quote.BidP)*float64(quote.BidV) + float64(quote.AskP)*float64(quote.AskV)
		pbook2 += float64(quote.BidV) + float64(quote.AskV)
	}
	return utils.SafeRatio(pbook1, pbook2, 0)
}

// CaliAVG 计算 档口index值平均,用于计算买档斜率(Bid Slope)
func CaliAVG(quotes []*msg.Quote) (iAVG int) {
	length := len(quotes)
	if length == 0 {
		return 0
	}
	sum := length * (length - 1) / 2
	return sum / length
}

// CalbidvAVG bidvAVG 为五档买量的算术平均值
func CalbidvAVG(quotes []*msg.Quote, quoteslength float64) (bidvAVG float64) {
	if quoteslength == 0 {
		return 0
	}
	for _, quote := range quotes {
		bidvAVG += float64(quote.BidV)
	}
	return bidvAVG / quoteslength
}

// CalaskvAVG askvAVG 为五档卖量的算术平均值
func CalaskvAVG(quotes []*msg.Quote, quoteslength float64) (askvAVG float64) {
	if quoteslength == 0 {
		return 0
	}
	for _, quote := range quotes {
		askvAVG += float64(quote.AskV)
	}
	return askvAVG / quoteslength
}

// CalSlopeBid 买档斜率(Bid Slope)
func CalSlopeBid(quotes []*msg.Quote, iAVG int, bidvAVG float64) (slopebid float64) {
	var slope_bid1 float64
	var slope_bid2 float64
	for i, quote := range quotes {
		slope_bid1 += float64(i-iAVG) * (float64(quote.BidV) - bidvAVG)
		slope_bid2 += float64(i-iAVG) * float64(i-iAVG)
	}
	return utils.SafeRatio(slope_bid1, slope_bid2, 0)
}

// CalSlopeAsk 卖档斜率(Ask Slope)
func CalSlopeAsk(quotes []*msg.Quote, iAVG int, askvAVG float64) (slopeask float64) {
	var slope_ask1 float64
	var slope_ask2 float64
	for i, quote := range quotes {
		slope_ask1 += float64(i-iAVG) * (float64(quote.BidV) - askvAVG)
		slope_ask2 += float64(i-iAVG) * float64(i-iAVG)
	}
	return utils.SafeRatio(slope_ask1, slope_ask2, 0)
}

// CalSpread 瞬时买卖价差（单位：BP 基点），涨停或跌停封板时分母为 0，此时价差置为最大值 9999
func CalSpread(askp0 float64, bidp0 float64) (spread float64) {
	if askp0 > 0 && bidp0 > 0 {
		return (askp0 - bidp0) / ((askp0 + bidp0) / 2) * 10000
	}
	return 9999
}

//---------------- Bar相关指标计算函数 ----------------//

// CalBodyRatio K 线实体比例,若 High_t = Low_t,即一字板，则 BodyRatio_t = 100%
func CalBodyRatio(open float64, close float64, high float64, low float64) (bodyratio float64) {
	return utils.SafeRatio(math.Abs(close-open), (high - low), 1)
}
