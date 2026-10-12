# 前端接口契约

本文件是前端页面（`frontend/fcinfo`）与后端之间的接口约定。前端 `src/types/stock.ts` 的类型与本文一一对应，改动需同步两边。

## 约定

| 项 | 约定 |
| --- | --- |
| base | `/api`，按服务分前缀：行情查询 `/api/market/*`、订阅与持仓 `/api/business/*`；鉴权的 `/login`、`/register` 挂在**根路径**下 |
| 时间 | 一律 ISO 8601 带时区，如 `2026-08-13T14:56:03+08:00`（A股时区 `Asia/Shanghai`），展示格式化由前端负责 |
| 数值 | 后端只返回原始数值，不下发 `4.82亿`、`+2.30%` 这类成品文案 |
| 错误体 | `{ "error": { "code": "...", "message": "..." } }` |
| 实时性 | 分时与快照 5s 轮询，日K 与持仓低频刷新；V1 全量返回，`since` 增量参数留 V2 |
| 鉴权 | JWT。`POST /login` 换 token（**这两个接口不在 `/api` 前缀下**，见第三节），之后所有 `/api/*` 带 `Authorization: Bearer <token>` |

前端实现：`src/api/request.ts`（统一请求层）、`src/api/market.ts`、`src/api/business.ts`、`src/api/auth.ts`（登录/注册）。

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

- **`prevClose` / `change` / `changePercent` 可以为 `null`** —— 库里只有今天的数据时（系统刚上线、或该 symbol 昨天没跑）没有昨收，涨跌幅就没有基准。此时三个字段一起回 `null`，前端据此隐藏这一行；**不要回 0**，那会让卡片显示 `+0.00 (0.00%)`，看着像"今天没涨没跌"。

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
| date | string | 否 | `YYYY-MM-DD HH:MM:SS`，缺省为当前时间。**注意格式与 `/tick` 的 `YYYY-MM-DD` 不同** |

> ⚠️ 后端的查询条件是 `created_at` 落在 **`date ± 7 天`** 的窗口里（不是"从 date 往前取"）。所以 `period=1d` 时最多只能拿到窗口内约 11 个交易日的日K，**不再是 30 根**；分钟级周期（`1m`/`5m`/`15m`）在这个窗口里取最近 `limit` 根是合理的。要让日K 能往回看更久，得把这个窗口放宽或者改成真正的区间参数。

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

请求体**只有 `code`**：取数路径由后端按代码决定，前端不参与选源（页面上原有的「数据来源」下拉已删掉，只有 API 一条路）。

订阅与查看是**两件事**：输入代码点「查看」只是切换当前展示的标的，会不会写进自选列表取决于用户有没有点「订阅」。

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

## 三、鉴权服务（根路径，无 `/api` 前缀）

这两个接口是 gin-jwt 自带的 handler，**响应体形状与本文其余部分不一致**，单独说明。

### 9. 登录

```
POST /login
{ "username": "alice", "password": "secret123" }
```

**响应** `200`

```json
{ "code": 200, "token": "<jwt>", "expire": "2026-10-12T15:04:05+08:00" }
```

`token` 是**裸 token，不带 `Bearer ` 前缀**，前端存下来后在请求头里拼成 `Authorization: Bearer <token>`。

**失败** `401`

```json
{ "code": 401, "message": "incorrect Username or Password" }
```

> ⚠️ 登录/注册用的是 `{ code, token, expire }` / `{ code, message }`，**不是上面约定的 `{ error: { code, message } }`** —— 这是 gin-jwt 的默认行为。前端 `request.ts` 的解析器两种都认，但**新接口不要再照着这个形状做**。

### 10. 注册

```
POST /register
{ "username": "alice", "password": "secret123" }
```

**响应** `201` —— 不回 id / username，前端拿到后用不上

```json
{ "code": 201 }
```

**失败**

| HTTP | 场景 |
| --- | --- |
| `400` `INVALID_PARAM` | 用户名或密码为空（gin 的 `binding:"required"`） |
| `409` | 用户名已存在 |

> 409 后端尚未实现：`RegisterHandler` 目前不检查用户名重复，直接 INSERT。`users.username` 也没有唯一约束，所以现在还拦不住。要真正生效得先加唯一索引再在 handler 里判。此外前端校验的「密码至少 6 位」后端也没有对应校验，只是前端 UX。

**注册不返回 token** —— 前端注册成功后切回登录页，让用户再走一次登录。

写入 `users` 表的约束（表结构见 `README.md:612`）：

- `password` 必须写 **bcrypt** 哈希（`bcrypt.GenerateFromPassword`），因为登录侧用的是 `bcrypt.CompareHashAndPassword`。README 里那句 `sha256+salt` 是过期描述
- 新用户的 `role` 给 `user`。**后端需同步把 `authorizator` 从只认 `role == "admin"` 放宽成"任意已登录用户"**，否则注册出来的账号能登录但调 `/api/*` 全 403
- `username` 在表里**没有唯一约束**，"用户名已存在"得后端自己查一次再决定返 409

---

## 四、错误码

| HTTP | code | 场景 | 前端处置 |
| --- | --- | --- | --- |
| 400 | `INVALID_PARAM` | keyword 为空、period 非法 | warning |
| 404 | `SYMBOL_NOT_FOUND` | 代码不存在 | warning「未找到」 |
| 404 | `SUBSCRIPTION_NOT_FOUND` | 取消不存在的订阅 | 静默刷新 |
| 409 | `ALREADY_SUBSCRIBED` | 重复订阅 | info「已在订阅列表中」 |
| 404 | `NO_DATA` | 该 symbol 无行情表 / 当日无数据 | 空态 |
| 401 | `UNAUTHORIZED` | 未登录或 token 过期 | 清 token + 跳登录 |
| 403 | `FORBIDDEN` | 已登录但角色不够（后端 `authorizator` 目前只放行 admin） | 提示无权限 |
| 500 | `INTERNAL` | 服务端异常 | error |

前端统一由 `src/api/request.ts` 解析，抛 `ApiRequestError(status, code, message)`。

> 例外：`/login`、`/register` 返回的是 `{ code, message }`（gin-jwt 的默认形状，见第三节）。`request.ts` 两种都认，但**只有这两个接口是例外**，新接口一律用上面的统一错误体。

---

## 五、落地备注

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

### 已知的后端问题（联调阻塞）

这三条前端绕不过去，必须后端改：

1. **登录按 `id` 查，不是按 `username`** —— `backend/common/auth/auth.go` 里是
   `SELECT id, username, password, role FROM users WHERE id=$1`，但绑定的是登录表单里的
   `username` 字符串。结果只有纯数字用户名能登录成功，非数字会因类型转换直接报错，前端只看到 401。
   **用户侧的表现是「刚注册的账号登不进去」。** 应改成 `WHERE username=$1`。
2. **`authorizator` 只放行 `role == "admin"`** —— 非 admin 账号能登录，但所有 `/api/*` 返回 403。
   登录响应里没有 role 字段，前端没法提前告知，只能在第一次业务请求吃到 403 时提示。
3. **`identityHandler` 的 `claims["id"].(int)` 会 panic** —— JWT 经 JSON 往返后数字是 `float64`，
   断言 `int` 失败 → panic → gin Recovery → **500 且 body 为空**，admin 也照样中。
   修法一行：`int(claims[identityKey].(float64))`。

排查命令（拿到 token 后打一个受保护接口）：

```bash
curl -i 'http://127.0.0.1:8555/api/market/stocks/search?keyword=1' \
  -H 'Authorization: Bearer <token>'
```

500 空 body 就是第 3 条，403 是第 2 条，401 是第 1 条。

### 部署注意

`/login`、`/register` 既是 SPA 路由又是后端接口。开发环境靠 `vite.config.ts` 里的 `bypass`
按 `Accept: text/html` 分流，**生产环境同样要按方法分流**（例如 nginx 里 `GET /login` 交给
`index.html`、`POST /login` 转发后端），否则刷新登录页会直接打到后端拿 404。

### 前端待办

- `src/mock/stockData.ts` 已无人引用（MainPage 早就切到 API 了），可以删 —— 它现在唯一的副作用是改类型时会连带编译失败
- `vite.config.ts` 的 `AutoImport` 里还留着 `imports: ['pinia']`，但项目没装 pinia —— 幽灵配置，建议去掉
- 退出登录只清本地 token，后端的 JWT 仍然有效直到过期（1h）。要做服务端失效需要黑名单，V1 不做
