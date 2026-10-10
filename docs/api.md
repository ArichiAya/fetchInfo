# 前端接口契约

本文件是前端页面（`frontend/fcinfo`）与后端之间的接口约定。前端 `src/types/stock.ts` 的类型与本文一一对应，改动需同步两边。

## 约定

| 项 | 约定 |
| --- | --- |
| base | `/api`，按服务分前缀：行情查询 `/api/market/*`、订阅与持仓 `/api/business/*` |
| 时间 | 一律 ISO 8601 带时区，如 `2026-08-13T14:56:03+08:00`（A股时区 `Asia/Shanghai`），展示格式化由前端负责 |
| 数值 | 后端只返回原始数值，不下发 `4.82亿`、`+2.30%` 这类成品文案 |
| 错误体 | `{ "error": { "code": "...", "message": "..." } }` |
| 实时性 | 分时与快照 5s 轮询，日K 与持仓低频刷新；V1 全量返回，`since` 增量参数留 V2 |
| 鉴权 | 未定。接入后前端需在 `src/api/request.ts` 补 token，并处理 401 |

前端实现：`src/api/request.ts`（统一请求层）、`src/api/market.ts`、`src/api/business.ts`。

---

## 一、行情查询服务 `/api/market`

### 1. 搜索标的

```
GET /api/market/stocks/search?keyword=101&limit=10
```

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| keyword | string | 是 | 代码，**模糊匹配** |
| limit | number | 否 | 默认 10 |

**响应** `200`

```json
{ "items": [{ "code": "000001", "name": "平安银行" }] }
```

V1 不返回现价：订阅输入框用不到，为此 join 实时行情不划算。

### 2. 行情快照

```
GET /api/market/stocks/{code}/snapshot
```

**响应** `200`

```json
{
  "code": "000001",
  "name": "平安银行",
  "ts": "2026-08-13T14:56:03+08:00",
  "price": 12.86,
  "prevClose": 12.57,
  "high": 12.92,
  "low": 12.51,
  "change": 0.29,
  "changePercent": 2.31,
  "cumVolume": 372100,
  "cumAmount": 482000000,
  "vwap": 12.80
}
```

字段来源：

- `price / high / low / cumVolume / cumAmount` — `msg.Tick` 同名字段
- `vwap` — `Tick2.vwap`（`indicator.CalVWAP`）
- `change / changePercent` — 后端算，需要 `prevClose`，见下
- **`prevClose`（昨收）** — `msg.Tick` 里没有，取"今天之前最近的一条 tick"的 `price`：

  ```sql
  SELECT tick_row->>'price' FROM {symbol}_Tick2
  WHERE created_at < $today ORDER BY created_at DESC LIMIT 1
  ```

  这样天然跳过周末与节假日，不需要交易日历。一个交易日内结果不变，可缓存。
  注意**不要用今开顶替**（A股有集合竞价，`open ≠ prevClose`），也不要查 Kafka（`bar2.new`/topic 保留期只有 1 天，周一与节后查不到）。

KPI 卡片映射：「现价」→ `price`+`changePercent`；「今日最高」→ `high`；「成交额」→ `cumAmount`；「持仓市值」→ 前端从持仓聚合，不单独出接口。

### 3. 分时数据

```
GET /api/market/stocks/{code}/tick?date=2026-08-13
```

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| date | string | 否 | 交易日 `YYYY-MM-DD`，缺省为当日 |

**响应** `200` — 一个交易日 240 个点（09:30–11:30、13:00–15:00）

```json
[{ "time": "09:30", "price": 12.86, "avgPrice": 12.80, "volume": 850 }]
```

两处口径必须与后端一致：

- **`avgPrice` 是 VWAP**（累计成交额 / 累计成交量），即 `indicator.CalVWAP` 的口径，**不是**"到当前点为止的算术均价"
- **`volume` 是该分钟的区间成交量**，不是累计量。用 `cum_volume[n] - cum_volume[n-1]` 求差；直接透传 `cum_volume` 会画出一条单调递增的柱子

数据由 tick 聚合到分钟（tick 每 3 秒一条快照，一分钟约 20 条）。

### 4. K 线数据

```
GET /api/market/stocks/{code}/kline?period=1d&limit=30
```

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| period | string | 否 | `1m` / `5m` / `15m` / `1d`，默认 `1d` |
| limit | number | 否 | 返回根数，默认 30 |

**响应** `200`

```json
[
  {
    "ts": "2026-08-13T00:00:00+08:00",
    "open": 12.60, "close": 12.86, "high": 12.92, "low": 12.51,
    "volume": 1200, "amount": 1536000
  }
]
```

- 时间字段用 `ts`（ISO 8601）而非 `date`，因为 `period` 已参数化，日K 与分钟K 共用同一结构，展示格式由前端按 `period` 决定
- MA5/MA10 由前端计算（`KLineChart.vue`），**不建指标接口**
- 日K 由 `{symbol}_Tick2` 聚合：
  - `open` = 当日第一个 tick 的 `tick_row.open`
  - `high` / `low` = 当日所有 tick 的 `tick_row.high` / `low` 取 max / min
  - `close` = 当日最后一个 tick 的 `tick_row.price`
  - `volume` / `amount` = 当日最后一个 tick 的 `cum_volume` / `cum_amount`
  - 不要自己从 tick 序列算 OHLC，tick 自带的 `open/high/low` 已是当日累计值，更准
- 聚合放在查询侧现算（30 天约 14 万行，TimescaleDB 毫秒级），暂不改 compute 预聚合
- **当日日K 是未完成的 bar**，最后一根会随行情变动
- **历史日K 聚合不出来**：`{symbol}_Tick2` 只有系统运行之后的数据。要凑满 30 天必须一次性从掘金 history 接口回填

---

## 二、订阅与持仓服务 `/api/business`

### 5. 已订阅列表

```
GET /api/business/subscriptions
```

**响应** `200`

```json
[{ "code": "000001", "name": "平安银行", "subscribedAt": "2026-08-01T09:15:00+08:00" }]
```

### 6. 订阅

```
POST /api/business/subscriptions
{ "code": "000001" }
```

请求体**只有 `code`**：数据来源由后端按代码决定，前端不参与选源（页面上「数据来源」下拉是纯前端状态，不入库）。

**响应** `201` — 返回订阅项，结构同列表元素。

失败：`404 SYMBOL_NOT_FOUND`、`409 ALREADY_SUBSCRIBED`

### 7. 取消订阅

```
DELETE /api/business/subscriptions/{code}
```

**响应** `204`，无响应体。失败：`404 SUBSCRIPTION_NOT_FOUND`

### 8. 持仓列表

```
GET /api/business/positions
```

**响应** `200`

```json
[{ "code": "600519", "name": "贵州茅台", "shares": 200, "cost": 1580.0, "price": 1665.8 }]
```

- 只返回 `shares > 0` 的行
- `price` 现价由后端 join 好，前端只算市值与盈亏（`calcPositionValue`）
- 持仓在交易账户（掘金 → `market/internal/gateway` → business 的 PostgreSQL），现价在行情侧。建议 market 维护一份"最新价"缓存（Redis / 内存），business 读缓存，不要直连 TimescaleDB 逐 symbol 查

---

## 三、错误码

| HTTP | code | 场景 | 前端处置 |
| --- | --- | --- | --- |
| 400 | `INVALID_PARAM` | keyword 为空、period 非法 | warning |
| 404 | `SYMBOL_NOT_FOUND` | 代码不存在 | warning「未找到」 |
| 404 | `SUBSCRIPTION_NOT_FOUND` | 取消不存在的订阅 | 静默刷新 |
| 409 | `ALREADY_SUBSCRIBED` | 重复订阅 | info「已在订阅列表中」 |
| 404 | `NO_DATA` | 该 symbol 无行情表 / 当日无数据 | 空态 |
| 401 | `UNAUTHORIZED` | 未登录或 token 过期 | 跳登录 |
| 500 | `INTERNAL` | 服务端异常 | error |

前端统一由 `src/api/request.ts` 解析，抛 `ApiRequestError(status, code, message)`。

---

## 四、落地备注

### 表名规则未定（头号阻塞）

`symbol` 取值为掘金格式（如 `SHSE.600519`，见 `backend/ingestion/subscribe.py`），落库表名按 symbol 拼，目前有三个互不相同的版本：

| 出处 | `SHSE.600519` 拼出来 |
| --- | --- |
| `README.md:578` 设计稿 | `SHSE.600519_Tick2`（带点，建表需引号） |
| `persister/main.go` 的 `fixTableName`（实际执行的） | `shse_600519_tick2`（转小写 + 点转下划线） |
| 库里实际存在的表 | `1_SHSE_600519_Tick2` |

查询接口按哪条规则拼表名，直接决定它能不能查到数据。**开工前必须统一**。

逐 symbol 表的列结构在 `README.md:578-613`，与 persister 的 INSERT 参数顺序一致：

- `[symbol]_Tick2`：`created_at, symbol, tick_row(jsonb), delta_V, vwap, p_book, slope_bid, slope_ask, spread`（`ret` 不建字段，查询时算）
- `[symbol]_Bar2`：`created_at, symbol, bar_row(jsonb), body_ratio`（`tr`、`delta_close` 不建字段，查询时算）

查询侧需容忍表不存在（symbol 从未落库）。

### bar 链路当前是断的

`compute` 里 `// TODO: 创建BarDurableCOnsumer` 尚未实现，ingestion 发到 NATS 的 `bar.new` 无人消费，Kafka 的 `bar2.new` 没有生产者，`{symbol}_Bar2` 表是空的。

因此上文的日K 与 `prevClose` **都走 `{symbol}_Tick2`**，不依赖 bar 链路。tick 链路是通的（ingestion → tick_stream → compute → `tick2.new` → persister）。

### 前端待办

- `vite.config.ts` 未配 `/api` 代理，接入后端时需补
- `src/mock/stockData.ts` 目前仍在为页面供数，后端就绪后替换为 `src/api` 调用
- 轮询建议封装 `usePolling`，绑 `visibilitychange` 在标签页隐藏时暂停
- 页面上的「数据来源」下拉对取数已无影响，考虑去掉或改语义（如导出格式）
