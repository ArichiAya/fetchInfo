// Package query 提供 TimescaleDB 只读查询层:按 (symbol, period, 时间范围)
// 取 kline / indicator,历史区间大时优先命中连续聚合物化视图。
//
// 只读连接,不在此写入;写入统一由 market/persister 完成。
package query

import (
	"context"
	"fmt"
	"time"

	"github.com/ArichiAya/fetch-info/common/msg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// func CacheInit() (*sync.Map, context.CancelFunc) {
// 	// 获取存在的symbol列表,用于查询时验证symbol合法性,避免无效查询。
// 	// 1. 连接 TimescaleDB
// 	// 2. 查询 symbol 列表,缓存到内存
// 	// 3. 定时从数据库加载 symbol 列表,并监听 symbol 表的变更事件,更新内存缓存。
// 	cache := &sync.Map{}
// 	ctx, cancel := context.WithCancel(context.Background())

// 	// 从数据库加载 symbol 列表,并缓存到内存
// 	conn, err := pgx.Connect(ctx, databaseURL)
// 	if err != nil {
// 		log.Fatalf("Fail to connect pg: %v", err)
// 	}
// 	rows, err := conn.Query(ctx, "SELECT symbol FROM subscribe")
// 	if err != nil {
// 		log.Fatalf("Fail to query symbols: %v", err)
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var symbol string
// 		if err := rows.Scan(&symbol); err != nil {
// 			log.Fatalf("Fail to scan symbol: %v", err)
// 		}
// 		cache.Store(symbol, struct{}{})
// 	}

// 	go func() {
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			default:
// 				// 	// 监听nats消息,更新symbol缓存
// 			}
// 		}
// 	}()

// 	return cache, cancel
// }

func FetchTickData(ctx context.Context, p *pgxpool.Pool, symbol string, start time.Time, end time.Time) ([]*msg.Tick2, error) {
	// 查看是否有订阅对于应的 symbol
	symbols, err := p.Query(ctx, "SELECT symbol FROM subscribe WHERE data_type = 'tick' AND symbol = $1", symbol)
	if err != nil {
		return nil, fmt.Errorf("Fail to query symbols: %v", err)
	}
	defer symbols.Close()

	for symbols.Next() {
		var s string
		if err := symbols.Scan(&s); err != nil {
			return nil, fmt.Errorf("Fail to scan symbol: %v", err)
		}
		if s == symbol {
			// 查询 tick 数据
			tableName := symbol + "_Tick2"
			queryStr := "SELECT * FROM " + pgx.Identifier{tableName}.Sanitize()
			safeQueryStr := fmt.Sprint(queryStr, " WHERE time >= $1 AND time <= $2")
			rows, err := p.Query(ctx, safeQueryStr, start, end)
			if err != nil {
				return nil, fmt.Errorf("Fail to query tick data: %v", err)
			}
			defer rows.Close()

			var tick2s []*msg.Tick2
			for rows.Next() {
				var tick2Data msg.Tick2
				var createdAt__ time.Time
				var symbol__ string
				if err := rows.Scan(&createdAt__, &symbol__, &tick2Data.TickRow, &tick2Data.DeltaV, &tick2Data.Vwap, &tick2Data.PBook, &tick2Data.SlopeBid, &tick2Data.SlopeAsk, &tick2Data.Spread); err != nil {
					return nil, fmt.Errorf("Fail to scan tick data: %v", err)
				}

				tick2s = append(tick2s, &tick2Data)
			}
			return tick2s, nil
		}
	}

	return nil, fmt.Errorf("Symbol not found: %s", symbol)
}

func FetchBarData(ctx context.Context, p *pgxpool.Pool, symbol string, start time.Time, end time.Time) ([]*msg.Bar2, error) {
	// 查看是否有订阅对于应的 symbol
	symbols, err := p.Query(ctx, "SELECT symbol FROM subscribe WHERE data_type = 'bar' AND symbol = $1", symbol)
	if err != nil {
		return nil, fmt.Errorf("Fail to query symbols: %v", err)
	}
	defer symbols.Close()

	for symbols.Next() {
		var s string
		if err := symbols.Scan(&s); err != nil {
			return nil, fmt.Errorf("Fail to scan symbol: %v", err)
		}
		if s == symbol {
			// 查询 bar 数据
			tableName := symbol + "_Bar2"
			queryStr := "SELECT * FROM " + pgx.Identifier{tableName}.Sanitize()
			safeQueryStr := fmt.Sprint(queryStr, " WHERE time >= $1 AND time <= $2")
			rows, err := p.Query(ctx, safeQueryStr, start, end)
			if err != nil {
				return nil, fmt.Errorf("Fail to query bar data: %v", err)
			}
			defer rows.Close()

			var bar2s []*msg.Bar2
			for rows.Next() {
				var bar2Data msg.Bar2
				var createdAt__ time.Time
				var symbol__ string
				if err := rows.Scan(&createdAt__, &symbol__, &bar2Data.BarRow, &bar2Data.BodyRatio); err != nil {
					return nil, fmt.Errorf("Fail to scan bar data: %v", err)
				}

				bar2s = append(bar2s, &bar2Data)
			}
			return bar2s, nil
		}
	}

	return nil, fmt.Errorf("Symbol not found: %s", symbol)
}
