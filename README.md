# Fetch Info

# Vue 3 + TypeScript + Element Plus + SCSS 项目初始化指南

本指南基于 **Vite + Vue 3 + TypeScript + Element Plus + SCSS** 现代化技术栈，提供从零搭建工程化中后台前端项目的完整规范流程。

---

## 目录结构预估

项目配置完成后，标准的工程化目录结构如下：

```text
my-vue-app/
├── src/
│   ├── assets/
│   │   └── styles/
│   │       └── variables.scss      # 全局 SCSS 变量与 Mixin
│   ├── components/                 # 通用业务组件
│   ├── App.vue                     # 根组件
│   ├── main.ts                     # 入口文件
│   └── env.d.ts                    # 环境类型声明
├── auto-imports.d.ts               # 插件自动生成的 API 类型文件
├── components.d.ts                 # 插件自动生成的 组件类型文件
├── index.html                      # HTML 模版
├── package.json                    # 依赖配置文件
├── tsconfig.json                   # TS 根配置
├── tsconfig.app.json               # 应用级 TS 配置
└── vite.config.ts                  # Vite 核心配置文件
```

### 步骤一：创建 Vite + Vue + TS 项目

打开终端，运行 Vue 官方脚手架初始化指令：

```
npm create vite@latest my-vue-app -- --template vue-ts
```

创建成功后，**必须先进入项目子目录**，后续所有命令均在该目录下执行：

```
cd my-vue-app
```

### 步骤二：安装核心与开发依赖

在项目根目录下，依次安装组件库、预处理器与按需自动导入插件：

```
# 1. 安装基础依赖与 Element Plus 核心库
npm install
npm install element-plus

# 2. 安装 Sass 预处理器、自动导入插件与 Node 类型定义（开发依赖）
npm install -D sass unplugin-auto-import unplugin-vue-components @types/node
```

### 步骤三：准备 SCSS 全局样式文件

创建目录 `src/assets/styles/`，并在其中新建 `variables.scss` 文件，用于集中管理主题颜色和全局 Mixin：

```
// src/assets/styles/variables.scss

// 1. 全局主题色彩变量
$primary-color: #409eff;
$success-color: #67c23a;
$warning-color: #e6a23c;
$danger-color: #f56c6c;
$bg-color-base: #f5f7fa;

// 2. 全局常用布局 Mixin
@mixin flex-center($direction: row) {
  display: flex;
  flex-direction: $direction;
  justify-content: center;
  align-items: center;
}
```

### 步骤四：配置 `vite.config.ts`

替换根目录下的 `vite.config.ts`，配置 **Element Plus 按需自动引入**、**`@` 路径别名** 以及 **SCSS 全局变量自动注入**：

```
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

export default defineConfig({
  plugins: [
    vue(),
    // 自动导入 Vue 核心 API (ref, reactive) 与 Element Plus 反馈组件 (ElMessage 等)
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia'],
      resolvers: [ElementPlusResolver()],
      dts: 'auto-imports.d.ts'
    }),
    // 自动导入 Element Plus 视图组件
    Components({
      resolvers: [ElementPlusResolver()],
      dts: 'components.d.ts'
    })
  ],
  resolve: {
    alias: {
      // 配置 @ 指向 src 目录
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  css: {
    preprocessorOptions: {
      scss: {
        api: 'modern-compiler', // 适配 Sass 最新规范 API
        additionalData: `@use "@/assets/styles/variables.scss" as *;` // 注意末尾分号不可省略
      }
    }
  }
})
```

### 步骤五：补充 TypeScript 配置

为了让编辑器和 TS 编译器能够识别 `@` 别名路径以及自动生成的组件类型文件，修改 `tsconfig.app.json`（或 `tsconfig.json`）：

```
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": [
	"src/**/*.ts",
    "src/**/*.tsx",
    "src/**/*.vue",
    "src/**/*.d.ts",
    "auto-imports.d.ts",
    "components.d.ts"
  ]
}
```

### 步骤六：验证测试与启动

修改 `src/App.vue` 文件，测试 Element Plus 组件与全局 SCSS 变量是否生效（**注意：无需手动 import 组件或 API**）：

```
<template>
  <div class="main-container">
    <el-card class="box-card">
      <h2>Vue 3 + TS + Element Plus 工程环境搭建成功！</h2>
      <el-button type="primary" @click="handleClick">点击测试 ElMessage</el-button>
    </el-card>
  </div>
</template>

<script setup lang="ts">
const handleClick = () => {
  // ElMessage 已经被自动按需导入，且享有 TS 类型提示
  ElMessage.success('Element Plus 消息提示成功！')
}
</script>

<style lang="scss" scoped>
.main-container {
  @include flex-center(column); // 调用全局 SCSS Mixin
  min-height: 100vh;
  background-color: $bg-color-base; // 使用全局 SCSS 变量

  .box-card {
    text-align: center;
    h2 {
      color: $primary-color; // 使用全局 SCSS 主题色变量
      margin-bottom: 20px;
    }
  }
}
</style>
```

在终端中启动本地开发服务器：

```
npm run dev
```





# 后端结构



```
backend/
├── go.work                            # workspace: common + market + analysis + business
│
├── common/                            # ── 共享库模块(只被 import,无入口)──
│   ├── go.mod                         # module fetchinfo/common
│   ├── msg/                           # message.proto 生成的 Go 代码(从 ingestion/pbs 迁出)
│   ├── natsx/                         # JetStream 连接 / durable consumer 封装
│   ├── kafkax/                        # Producer/Consumer 封装,主题名常量集中在此
│   ├── pg/  redis/                    # pgx 连接池 / go-redis 客户端
│   ├── timescale/                     # TimescaleDB 连接与批次写入助手
│   └── auth/                          # 登录/JWT,Redis 存 session(带TTL)
│
├── ingestion/                         # ── ① [Python] 掘金数据订阅(现有)
│   └── subscribe.py                   # SDK订阅 → NATS(tick.> / bar.>)
│
├── market/                            # ── ②③⑤ 行情流水线(Go 模块,3 个入口)──
│   ├── go.mod
│   ├── cmd/
│   │   ├── compute/main.go            # 服务: NATS消费 → 计算指标 → Kafka
│   │   ├── persister/main.go          # 服务: Kafka消费 → TimescaleDB 批量写
│   │   └── trader/main.go             # 服务: 实时条件引擎 → 掘金交易
│   └── internal/
│       ├── indicator/                 # 指标计算
│       ├── model/                     # Tick2/Bar2 计算指标后的结构体
│       ├── strategy/                  # 交易条件定义/求值(被 trader 和 analysis 复用)
│       └── gateway/                   # 掘金交易 SDK 适配(下单/撤单/持仓回报)
│
├── analysis/                          # ── ④ 分析/报表 API(Go 模块)──
│   ├── go.mod
│   ├── cmd/server/main.go
│   └── internal/
│       ├── query/                     # TimescaleDB 只读查询(连续聚合加速报表)
│       ├── report/                    # 报表生成(csv/json/图表数据)
│       ├── trigger/                   # 手动分析请求 → 注册/触发交易条件
│       └── httpapi/                   # 路由与 handler
│
└── business/                          # ── ⑥ 业务服务(Go 模块)──
    ├── go.mod
    ├── cmd/server/main.go
    └── internal/
        ├── workorder/                 # 工单(PostgreSQL CRUD)
        ├── personnel/                 # 人员(PostgreSQL CRUD)
        └── httpapi/

```



数据流向

```
数据流水线

掘金数据 ➔ 掘金sdk数据订阅 ➔ NATS ➔ 降采样&计算指标 ➔ kafka ➔ TimescaleDB持久化

分析与交易触发

TimescaleDB持久化 ↔ api分析请求

api分析请求 ➔ 数据报表

api分析请求 ➔ 交易触发(满足条件时)

业务服务

业务api ↔ postgresql
```



# Tick指标

## 指标一：累积主动买卖量差（CIS）

**介绍**：衡量过去一段时间内，主动向上吃单（买入）与主动向下砸单（卖出）的成交量净额。该指标消除了价格的随机波动，直接反映**资金介入的强度**，与未来 1-2 小时的价格趋势呈显著正相关。

【Tick 原子判定逻辑】

- 若 $price \ge ask\_p_0$ 且 $ask\_p_0 > 0$，则 $\Delta V_t = +last\_volume$（主动买入）
- 若 $price \le bid\_p_0$ 且 $bid\_p_0 > 0$，则 $\Delta V_t = -last\_volume$（主动卖出）
- 若 $bid\_p_0 < price < ask\_p_0$ 或涨跌停边界，则 $\Delta V_t = 0$

【聚合窗口计算（长 Duration 因子）】
$$
CIS_{Window} = \sum_{t=1}^{N} \Delta V_t
$$


  ## 指标二：加权平均挂单价偏离度（VWAP & 订单簿重心偏离）

  **介绍**：VWAP（日内成交均价）代表全体交易者的平均成本，订单簿重心代表当前盘口挂单的加权成本。两者的偏离度用于判断**当前价格是否处于“高估/低估”区域**，是长线持仓中动态止盈止损的核心锚点。

  【Tick 原子指标】
  瞬时 VWAP（直接使用累计值）：
$$
  VWAP_t = \frac{cum\_amount}{cum\_volume}
$$

  五档订单簿重心（盘口加权价）：
$$
  P\_Book_t = \frac{\sum_{i=0}^{4} (bid\_p_i \times bid\_v_i + ask\_p_i \times ask\_v_i)}{\sum_{i=0}^{4} (bid\_v_i + ask\_v_i)}
$$
  *（注：涨跌停时对应缺失项为 0，分母自动剔除无效档位）*

  【聚合窗口计算（长 Duration 因子）】
  当前价格对 VWAP 的瞬时偏离度：
$$
  Deviation_{close} = \frac{price_{current} - VWAP_t}{VWAP_t} \times 100\%
$$

  窗口内平均订单簿重心偏离度：
$$
  Book\_Spread_{avg} = \frac{1}{N} \sum_{t=1}^{N} \left( \frac{P\_Book_t - VWAP_t}{VWAP_t} \times 100\% \right)
$$


  ## 指标三：订单簿斜率（Order Book Slope）

  **介绍**：计算买档（或卖档）挂单量随价格远离的衰减速度。斜率越陡峭（绝对值越大），说明主力将单量高度集中在最优价附近，防守/进攻意愿极强；斜率平坦则说明挂单分散。该指标用于预判**未来支撑位的坚固程度**

  【Tick 原子指标】
  买档斜率（Bid Slope），自变量 $X$ 为档位索引 $[0,1,2,3,4]$，因变量 $Y$ 为挂单量 $[bid\_v_0, ..., bid\_v_4]$：
$$
  Slope\_Bid_t = \frac{\sum_{i=0}^{4} (i - \bar{i})(bid\_v_i - \bar{V}_{bid})}{\sum_{i=0}^{4} (i - \bar{i})^2}
$$
  其中 $\bar{i} = 2$，$\bar{V}_{bid}$ 为五档买量的算术平均值。

  卖档斜率（Ask Slope）同理：
$$
  Slope\_Ask_t = \frac{\sum_{i=0}^{4} (i - \bar{i})(ask\_v_i - \bar{V}_{ask})}{\sum_{i=0}^{4} (i - \bar{i})^2}
$$
  其中 $\bar{i} = 2$，$\bar{V}_{ask}$ 为五档卖量的算术平均值。

  【聚合窗口计算（长 Duration 因子）】
  窗口内平均买档斜率与卖档斜率：
$$
  Slope\_Bid_{avg} = \frac{1}{N} \sum_{t=1}^{N} Slope\_Bid_t
$$
$$
  Slope\_Ask_{avg} = \frac{1}{N} \sum_{t=1}^{N} Slope\_Ask_t
$$


  ## 指标四：已实现波动率与价差扩张率（RV & Spread Expansion）

  **介绍**：RV 衡量价格波动的剧烈程度，价差衡量市场流动性充裕度。两者结合用于判断**“变盘临界点”**。当 RV 持续压缩至极低水平（平静期），突然伴随价差急剧扩张，往往是趋势启动的前兆。

  【Tick 原子指标】
  瞬时收益率（首个 Tick 置为 0）：
$$
  Ret_t = \frac{price_t - price_{t-1}}{price_{t-1}}
$$

  瞬时买卖价差（单位：BP 基点），涨停或跌停封板时分母为 0，此时价差置为最大值 9999：
$$
Spread_t = \frac{ask\_p_0 - bid\_p_0}{(ask\_p_0 + bid\_p_0) / 2} \times 10000
$$

  【聚合窗口计算（长 Duration 因子）】
  已实现波动率（窗口内收益率的标准差）：
$$
  RV_{Window} = \sqrt{ \frac{1}{N-1} \sum_{t=1}^{N} (Ret_t - \overline{Ret})^2 }
$$
  其中 $\overline{Ret}$ 为窗口内收益率的算术平均值。

  窗口内平均价差：
$$
  Avg\_Spread = \frac{1}{N} \sum_{t=1}^{N} Spread_t
$$

  价差扩张率（当前窗口平均价差 / 过去 20 个窗口平均价差的移动平均）：
$$
  Spread\_Expansion = \frac{Avg\_Spread}{MA\_20(Avg\_Spread)}
$$



| 字段名      | 对应指标                 | 类型            |
| :---------- | :----------------------- | :-------------- |
| `delta_v`   | 指标一（CIS）            | int（正/负/零） |
| `vwap`      | 指标二（瞬时 VWAP）      | float           |
| `p_book`    | 指标二（瞬时订单簿重心） | float           |
| `slope_bid` | 指标三（买档斜率）       | float           |
| `slope_ask` | 指标三（卖档斜率）       | float           |
| `ret`       | 指标四（瞬时收益率）     | float           |
| `spread`    | 指标四（瞬时价差 BP）    | float           |





# Bar指标

## 第一部分：单个 Bar 公式（不涉及窗口聚合）

### 1. 真实波幅（TR）

**介绍**：衡量当前 Bar 的价格波动范围，同时考虑 Bar 内振幅和隔夜跳空。作为后续 ATR 聚合计算的原料。

【单个 Bar 公式】
$$
TR_t = \max(High_t - Low_t,\; |High_t - Close_{t-1}|,\; |Low_t - Close_{t-1}|)
$$

---

### 2. K 线实体比例

**介绍**：衡量当前 Bar 收盘价与开盘价的差距占整个 Bar 波动的比例，用于判断多空力量的悬殊程度。

【单个 Bar 公式】
$$
BodyRatio_t = \frac{|Close_t - Open_t|}{High_t - Low_t} \times 100\%
$$
*（若 $High_t = Low_t$，即一字板，则 $BodyRatio_t = 100\%$）*

---

### 3. 收盘价变化量

**介绍**：当前 Bar 收盘价相对于前一个 Bar 收盘价的差值，作为 RSI 和 OBV 等聚合指标的计算原料。

【单个 Bar 公式】
$$
\Delta Close_t = Close_t - Close_{t-1}
$$


  ## 第二部分：聚合公式（需要 N 个 Bar 的窗口数据）

  ### 1. 简单移动平均（SMA）

  **介绍**：窗口内所有收盘价的算术平均值。

  【聚合公式】（窗口长度 N）
$$
  SMA_t(N) = \frac{1}{N} \sum_{i=0}^{N-1} Close_{t-i}
$$

---

  ### 2. 指数移动平均（EMA）

  **介绍**：对窗口内价格赋予指数递减权重，近期价格权重更高。

  【聚合公式】（窗口长度 N）
$$
  \alpha = \frac{2}{N + 1}
$$

$$
  EMA_t(N) = Close_t \times \alpha + EMA_{t-1}(N) \times (1 - \alpha)
$$

---

  ### 3. MACD 柱状线

  **介绍**：由快慢均线离差（DIF）及其平滑（DEA）的差值构成，反映动能的强弱和方向变化。

  【聚合公式】（窗口参数固定为 12、26、9）
$$
  DIF_t = EMA_t(12) - EMA_t(26)
$$

$$
  DEA_t = EMA_t(9) \quad \text{（对 DIF 序列做 EMA 计算）}
$$

$$
  MACD\_Bar_t = (DIF_t - DEA_t) \times 2
$$

---

  ### 4. 平均真实波幅（ATR）

  **介绍**：窗口内真实波幅（TR）的移动平均值，用于衡量市场平均波动强度。

  【聚合公式】（窗口长度 N = 14）
$$
  ATR_t(14) = \frac{1}{14} \sum_{i=0}^{13} TR_{t-i}
$$
  *（注：实际应用中常采用 Wilder 递推平滑算法，但此处给出标准算术平均定义）*

---

  ### 5. 布林带（Bollinger Bands）与带宽

  **介绍**：以移动平均线为中轨，加减若干倍标准差构成上下轨。带宽用于衡量波动率的相对高低。

  【聚合公式】（窗口长度 N = 20，倍数 K = 2）
$$
  Middle_t = SMA_t(20) \quad \text{（即 } Close \text{ 的 20 周期简单移动平均）}
$$

$$
  Std_t = \sqrt{\frac{1}{20} \sum_{i=0}^{19} (Close_{t-i} - Middle_t)^2}
$$

$$
  Upper_t = Middle_t + 2 \times Std_t
$$

$$
  Lower_t = Middle_t - 2 \times Std_t
$$

$$
  Bandwidth_t = \frac{Upper_t - Lower_t}{Middle_t} \times 100\% = \frac{4 \times Std_t}{Middle_t} \times 100\%
$$

---

  ### 6. 相对强弱指数（RSI）

  **介绍**：衡量窗口内价格变动的平均涨幅与平均跌幅之比，用于评估超买超卖状态。

  【聚合公式】（窗口长度 N = 14）
$$
  AvgGain_t = \frac{1}{14} \sum_{i=0}^{13} \max(\Delta Close_{t-i},\; 0)
$$

$$
  AvgLoss_t = \frac{1}{14} \sum_{i=0}^{13} \max(-\Delta Close_{t-i},\; 0)
$$

  *（注：$\Delta Close$ 为第一部分定义的单 Bar 变化量）*
$$
  RS_t = \frac{AvgGain_t}{AvgLoss_t}
$$

$$
  RSI_t = 100 - \frac{100}{1 + RS_t}
$$

  *（当 $AvgLoss_t = 0$ 时，$RSI_t = 100$）*

---

  ### 7. 能量潮（OBV）

  **介绍**：根据收盘价涨跌方向，对成交量进行累加或累减，用于验证价格走势是否获得成交量配合。

  【聚合公式】（递推形式，依赖前值）
$$
  OBV_t = 
  \begin{cases}
  OBV_{t-1} + Volume_t, & \text{if } Close_t > Close_{t-1} \\
  OBV_{t-1} - Volume_t, & \text{if } Close_t < Close_{t-1} \\
  OBV_{t-1}, & \text{if } Close_t = Close_{t-1}
  \end{cases}
$$
  *（注：$OBV_0$ 初始值为 0，$Volume_t$ 为当前 Bar 的成交量）*





# 数据库结构



subscribe

| symbol    | character varying |          |
| --------- | ----------------- | -------- |
| data_type | character varying |          |
| starttime | timestamp         | not null |

primary key: symbol+data_type





[symbol]_Tick2 //[symbol]为subscribe里的symbol值

| created_at | timestamp         |                                                          |
| ---------- | ----------------- | -------------------------------------------------------- |
| symbol     | character varying |                                                          |
| tick_row   | jsonb             | not null                                                 |
| delta_V    | bigint            |                                                          |
| vwap       | numeric           |                                                          |
| p_book     | numeric           |                                                          |
| slope_bid  | numeric           |                                                          |
| slope_ask  | numeric           |                                                          |
| ret        | numeric           | 不创建该字段,在查询时再计算:需要上一个tick的数据进行计算 |
| spread     | numeric           |                                                          |

primary key: created_at

不设置外键,避免设置了级联删除,删除关联数据时把该数据也删除了



[symbol]_Bar2 //[symbol]为subscribe里的symbol值

| created_at  | timestamp         |                                                         |
| ----------- | ----------------- | ------------------------------------------------------- |
| symbol      | character varying |                                                         |
| bar_row     | jsonb             | not null                                                |
| tr          | numeric           | 不创建该字段,在查询时再计算:需要上一个bar的数据进行计算 |
| body_ratio  | numeric           |                                                         |
| delta_close | numeric           | 不创建该字段,在查询时再计算:需要上一个bar的数据进行计算 |

primary key: created_at



users

| id       | bigint            | GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY |
| -------- | ----------------- | -------------------------------------------- |
| username | character varying | not null                                     |
| password | character varying | not null,sha256+salt                         |
| role     | character varying | not null                                     |
|          |                   |                                              |

