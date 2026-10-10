// Package httpapi 组装分析服务的 HTTP 层:路由、handler、响应格式。
//
// 路由规划(示例):
//
// 鉴权中间件校验 common/auth 签发的 JWT,登录态在 Redis 中校验。
package httpapi

import (
	"context"

	"github.com/ArichiAya/fetch-info/common/msg"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GET /api/market/stocks/search?keyword=101&limit=10
// | 参数    | 类型   | 必填 | 说明               |
// | ------- | ------ | ---- | ------------------ |
// | keyword | string | 是   | 代码，**模糊匹配** |
// | limit   | number | 否   | 默认 10            |
func SearchStocksHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 解析参数
		keyword := c.Query("keyword")
		limit := c.DefaultQuery("limit", "10")

		// 2. 查询TimescaleDB
		queryStr := `SELECT symbol FROM subscribe WHERE symbol ILIKE $1 LIMIT $2`
		rows, err := pool.Query(c.Request.Context(), queryStr, keyword, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query database"})
			return
		}
		defer rows.Close()

		var symbols []string
		for rows.Next() {
			var symbol string
			if err := rows.Scan(&symbol); err != nil {
				c.JSON(500, gin.H{"error": "Failed to scan row"})
				return
			}
			symbols = append(symbols, symbol)
		}

		// 3. 返回 JSON 响应
		c.JSON(200, gin.H{"symbols": symbols})
	}
}

// GET /api/market/stocks/{code}/snapshot
//
//	{
//	  "code": "000001",
//	  "name": "平安银行",
//	  "ts": "2026-08-13T14:56:03+08:00",
//	  "price": 12.86,
//	  "prevClose": 12.57,
//	  "high": 12.92,
//	  "low": 12.51,
//	  "change": 0.29,
//	  "changePercent": 2.30,
//	  "cumVolume": 372100,
//	  "cumAmount": 482000000,
//	  "vwap": 12.80
//	}
func GetStockSnapshotHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")

		// 检查 code 是否存在
		queryStr := `SELECT symbol FROM subscribe WHERE symbol = $1`
		row := pool.QueryRow(c.Request.Context(), queryStr, code)
		var symbol string
		if err := row.Scan(&symbol); err != nil {
			c.JSON(404, gin.H{"error": "Stock code not found"})
			return
		}

		// 查询最新的快照数据
		tableName := symbol + "_Tick2"
		queryStr = `SELECT created_at,tick_row,vwap from ` + tableName + ` ORDER BY created_at DESC LIMIT 2`
		safeQueryStr := pgx.Identifier{queryStr}.Sanitize() // 防止SQL注入
		rows, err := pool.Query(c.Request.Context(), safeQueryStr)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query database"})
			return
		}
		defer rows.Close()

		var snapshots []*msg.Tick
		var vwaps []float64
		for rows.Next() {
			var createdAt string
			var snapshot msg.Tick
			var vwap float64
			if err := rows.Scan(&createdAt, &snapshot, &vwap); err != nil {
				c.JSON(500, gin.H{"error": "Failed to scan row"})
				return
			}
			snapshots = append(snapshots, &snapshot)
			vwaps = append(vwaps, vwap)
		}

		if len(snapshots) == 0 {
			c.JSON(404, gin.H{"error": "No snapshot data found"})
			return
		}

		// 计算 change 和 changePercent
		latest := snapshots[0]
		prev := snapshots[1]
		change := latest.Price - prev.Price
		changePercent := (change / prev.Price) * 100

		response := gin.H{
			"code":          latest.Symbol,
			"name":          latest.Symbol,
			"ts":            latest.CreatedAt,
			"price":         latest.Price,
			"prevClose":     prev.Price,
			"high":          latest.High,
			"low":           latest.Low,
			"change":        change,
			"changePercent": changePercent,
			"cumVolume":     latest.CumVolume,
			"cumAmount":     latest.CumAmount,
			"vwap":          vwaps[0],
		}

		c.JSON(200, response)
	}
}

// GET /api/market/stocks/{code}/tick?date=2026-08-13
// | 参数 | 类型   | 必填 | 说明                            |
// | ---- | ------ | ---- | ------------------------------- |
// | date | string | 否   | 交易日 `YYYY-MM-DD`，缺省为当日 |
// **响应** `200` — 一个交易日 240 个点（09:30–11:30、13:00–15:00）
// [{ "time": "09:30", "price": 12.86, "avgPrice": 12.80, "volume": 850 }]
// 两处口径必须与后端一致：

// - **`avgPrice` 是 VWAP**（累计成交额 / 累计成交量），即 `indicator.CalVWAP` 的口径，**不是**"到当前点为止的算术均价"
// - **`volume` 是该分钟的区间成交量**，不是累计量。用 `cum_volume[n] - cum_volume[n-1]` 求差；直接透传 `cum_volume` 会画出一条单调递增的柱子

// 数据由 tick 聚合到分钟（tick 每 3 秒一条快照，一分钟约 20 条）。
func GetStockTickHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		date := c.DefaultQuery("date", "")

		// 检查 code 是否存在
		queryStr := `SELECT symbol FROM subscribe WHERE symbol = $1`
		row := pool.QueryRow(c.Request.Context(), queryStr, code)
		var symbol string
		if err := row.Scan(&symbol); err != nil {
			c.JSON(404, gin.H{"error": "Stock code not found"})
			return
		}

		tableName := symbol + "_Tick2"
		// 用plv8解析jsonb，提取出tick_row中的字段，按分钟聚合
		queryStr = `
		WITH ranked_data AS (
			SELECT time_bucket('minute',created_at) minute,price,ROW_NUMBER() OVER (PARTITION BY time_bucket('minute',created_at) ORDER BY created_at DESC) rn,
			sum(get_tick_row_cum_amount(tick_row)::numeric) over(PARTITION BY time_bucket('minute',created_at)) cum_amount,
			sum(get_tick_row_cum_volume(tick_row)::numeric) over(PARTITION BY time_bucket('minute',created_at)) cum_volume
			FROM ` + tableName + `
			WHERE time_bucket('day',created_at) = $1::date
		)
		SELECT minute,price,cum_amount/cum_volume avg_price,cum_volume - lag(cum_volume,1) over (order by minute) volume
		FROM ranked_data
		WHERE rn = 1
		ORDER BY minute
		`
		safeQueryStr := pgx.Identifier{queryStr}.Sanitize()
		rows, err := pool.Query(context.Background(), safeQueryStr, date)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query database"})
			return
		}
		defer rows.Close()

		var ticks []gin.H
		for rows.Next() {
			var minute string
			var price float64
			var avgPrice float64
			var volume float64
			if err := rows.Scan(&minute, &price, &avgPrice, &volume); err != nil {
				c.JSON(500, gin.H{"error": "Failed to scan row"})
				return
			}
			ticks = append(ticks, gin.H{
				"time":     minute,
				"price":    price,
				"avgPrice": avgPrice,
				"volume":   volume,
			})
		}

		c.JSON(200, ticks)
	}
}

// GET /api/market/stocks/{code}/kline?period=1d&limit=30
// | 参数   | 类型   | 必填 | 说明                                  |
// | ------ | ------ | ---- | ------------------------------------- |
// | period | string | 否   | `1m` / `5m` / `15m` / `1d`，默认 `1d` |
// | limit  | number | 否   | 返回根数，默认 30                     |
// [
//
//	{
//	  "ts": "2026-08-13T00:00:00+08:00",
//	  "open": 12.60, "close": 12.86, "high": 12.92, "low": 12.51,
//	  "volume": 1200, "amount": 1536000
//	}
//
// ]
func GetStockKlineHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		period := c.DefaultQuery("period", "1d")
		limit := c.DefaultQuery("limit", "30")

		// 检查 code 是否存在
		queryStr := `SELECT symbol FROM subscribe WHERE symbol = $1`
		row := pool.QueryRow(c.Request.Context(), queryStr, code)
		var symbol string
		if err := row.Scan(&symbol); err != nil {
			c.JSON(404, gin.H{"error": "Stock code not found"})
			return
		}

		tableName := symbol + "_Tick2"
		queryStr = `
		WITH ranked_data AS (
			SELECT time_bucket($1,created_at) period,price,ROW_NUMBER() OVER (PARTITION BY time_bucket($1,created_at) ORDER BY created_at DESC) rn,
			sum(get_tick_row_cum_amount(tick_row)::numeric) over(PARTITION BY time_bucket($1,created_at)) cum_amount,
			sum(get_tick_row_cum_volume(tick_row)::numeric) over(PARTITION BY time_bucket($1,created_at)) cum_volume,
			max(price) over(PARTITION BY time_bucket($1,created_at)) high,
			min(price) over(PARTITION BY time_bucket($1,created_at)) low,
			first_value(price) over(PARTITION BY time_bucket($1,created_at) ORDER BY created_at ASC) open
			FROM ` + tableName + `
		)
		SELECT period,open,price close,high,low,cum_volume volume,cum_amount amount
		FROM ranked_data
		WHERE rn = 1
		ORDER BY period
		LIMIT $2
		)
		`
		safeQueryStr := pgx.Identifier{queryStr}.Sanitize()
		rows, err := pool.Query(context.Background(), safeQueryStr, period, limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to query database"})
			return
		}
		defer rows.Close()

		var klines []gin.H
		for rows.Next() {
			var period string
			var open, close, high, low, volume, amount float64
			if err := rows.Scan(&period, &open, &close, &high, &low, &volume, &amount); err != nil {
				c.JSON(500, gin.H{"error": "Failed to scan row"})
				return
			}
			klines = append(klines, gin.H{
				"ts":     period,
				"open":   open,
				"close":  close,
				"high":   high,
				"low":    low,
				"volume": volume,
				"amount": amount,
			})
		}

		c.JSON(200, klines)
	}
}
