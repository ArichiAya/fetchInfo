/**
 * 股票行情相关接口定义
 * 契约见仓库根目录 docs/api.md；与后端对齐后，把 mock 换成 API 请求即可
 */

// ---------- 行情 ----------

/** 分时数据点（每分钟一个，一个交易日 240 个） */
export interface TickPoint {
  /** 时间，如 '09:30' */
  time: string
  /** 当前价 */
  price: number
  /**
   * 均价线：VWAP = 累计成交额 / 累计成交量
   * 口径与后端 indicator.CalVWAP 一致（不是"到当前点为止的算术均价"）
   */
  avgPrice: number
  /** 该分钟成交量（手），是区间量而非累计量 */
  volume: number
}

/** K 线数据（默认日K） */
export interface KLineDay {
  /**
   * 时间，ISO 8601 带时区，如 '2026-08-13T00:00:00+08:00'
   * 展示格式由前端按 period 决定（日K 显示日期，分钟K 显示时分）
   */
  ts: string
  open: number
  close: number
  high: number
  low: number
  /** 成交量（手） */
  volume: number
  /** 成交额（元） */
  amount: number
}

/** 行情快照（KPI 卡片的数据源，字段为原始数值，格式化由前端负责） */
export interface Snapshot {
  code: string
  name: string
  /** 服务端时间，ISO 8601 */
  ts: string
  /** 现价 */
  price: number
  /**
   * 昨收（上一交易日收盘价）
   * 库里只有今天的数据时为 null（系统刚上线，没有昨收可作基准）
   */
  prevClose: number | null
  high: number
  low: number
  /** 涨跌额 = price - prevClose，无昨收时为 null */
  change: number | null
  /** 涨跌幅 %，无昨收时为 null */
  changePercent: number | null
  /** 当日累计成交量（手） */
  cumVolume: number
  /** 当日累计成交额（元） */
  cumAmount: number
  /** 当日均价 */
  vwap: number
}

/** 标的基础信息（搜索结果 / 订阅目录） */
export interface StockSymbol {
  code: string
  name: string
}

// ---------- 组合 ----------

/** 持仓项（现价由后端 join 好，前端只算市值与盈亏） */
export interface PositionItem {
  code: string
  name: string
  /** 持股数量 */
  shares: number
  /** 成本价 */
  cost: number
  /** 现价 */
  price: number
}

/** 持仓项（含市值，由前端计算） */
export interface PositionItemWithValue extends PositionItem {
  /** 市值 = shares * price */
  marketValue: number
  /** 盈亏额 */
  profit: number
  /** 盈亏比例 % */
  profitPercent: number
}

/** 订阅项 */
export interface SubscriptionItem {
  code: string
  name: string
  /** 订阅时间，ISO 8601 带时区 */
  subscribedAt: string
}

// ---------- 纯前端模型（不入后端） ----------

/**
 * KPI 指标卡片
 * 由 snapshot + 持仓聚合在前端拼装，后端只返回数值
 */
export interface KpiItem {
  label: string
  value: string
  /** 与上一周期相比的变化说明，如 '+0.29 (2.30%)' */
  delta?: string
  /** 变化方向是否利好（红色/绿色由涨跌决定） */
  up?: boolean
  /** 是否为主展示数字（hero，加大字号） */
  hero?: boolean
}

/** 数据来源选项（纯前端状态，不参与后端取数） */
export interface DataSourceOption {
  label: string
  value: string
}

// ---------- 错误 ----------

/** 后端统一错误体：{ "error": { "code": "...", "message": "..." } } */
export interface ApiErrorBody {
  error: {
    code: ApiErrorCode | string
    message: string
  }
}

export type ApiErrorCode =
  /** 入参非法（keyword 为空、period 非法等） */
  | 'INVALID_PARAM'
  /** 股票代码不存在 */
  | 'SYMBOL_NOT_FOUND'
  /** 取消订阅时该订阅不存在 */
  | 'SUBSCRIPTION_NOT_FOUND'
  /** 重复订阅 */
  | 'ALREADY_SUBSCRIBED'
  /** 该 symbol 无行情表 / 当日无数据 */
  | 'NO_DATA'
  /** 未登录或 token 过期 */
  | 'UNAUTHORIZED'
  | 'INTERNAL'
