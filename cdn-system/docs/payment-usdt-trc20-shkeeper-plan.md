# USDT(TRC20) 充值与套餐自动化 - SHKeeper 落地方案

## 1. 目标与范围

本方案用于在现有 `order + user.balance + user_package` 体系上，补齐可运营的充值/开通/续费链路：

1. 用户可发起 USDT(TRC20) 充值订单。
2. 使用 SHKeeper 创建收款请求，并获取支付地址/金额。
3. SHKeeper 回调后自动入账余额并记流水。
4. 支持管理员调试：订单手动标记已付款（无真实支付时可验证套餐逻辑）。
5. 支持管理员手动加减余额，且必须写订单和余额流水。
6. 后续支持“余额自动续费套餐”（本阶段设计并预留接口，下一阶段落地 worker）。

---

## 2. 选型结论（SHKeeper 是否合适）

结论：**适合当前阶段接入**，但建议按“支付适配层”落地，避免未来被单一网关绑定。

### 2.1 优点

1. 开源、自托管，适合你当前私有部署形态。
2. 支持 `USDTTRC20` 支付方式，满足当前主诉求。
3. 提供 HTTP API（创建发票、查询订单）+ 回调机制，便于自动入账。
4. 回调支持 `X-Shkeeper-Api-Key`，可以做服务端来源校验。

### 2.2 风险与约束

1. 回调可靠性依赖网络与幂等处理，必须做到“重复通知不重复入账”。
2. 汇率波动与链上手续费模型要前置定义（是否固定法币金额、超付/少付策略）。
3. 回调是公网入口，必须限制源校验、请求体校验、失败告警。
4. 资金类功能必须全程事务化，不能只靠前端状态。

---

## 3. 总体架构

```text
User -> API(创建充值单) -> SHKeeper(create invoice)
                          -> 返回 address/amount/network/orderNo

User 支付链上 USDT
SHKeeper -> API 回调(/api/v1/pay/shkeeper/callback)
          -> 校验 api key + 订单幂等
          -> 更新 order=paid
          -> 增加 user.balance
          -> 写 balance_ledger
```

---

## 4. 数据设计

## 4.1 复用现有表

1. `order`：作为统一业务订单（充值/开通/续费）。
2. `user.balance`：余额主字段。
3. `balance_ledger`：余额变更审计（已新增）。

## 4.2 order.data JSON 扩展（建议）

用于支付网关字段扩展，不强行新增表：

```json
{
  "channel": "shkeeper",
  "network": "trc20",
  "currency": "USDT",
  "shkeeper_invoice_id": "xxxx",
  "payment_url": "https://...",
  "address": "TRX....",
  "expected_amount": "12.34",
  "requested_fiat_amount": "100.00",
  "requested_fiat_currency": "CNY",
  "callback_payload": {}
}
```

---

## 5. 配置项设计（system/global）

建议落在 `config` 表（`type=system scope=global`）：

1. `pay_shkeeper_enable`：是否启用 SHKeeper。
2. `pay_shkeeper_base_url`：SHKeeper API 基地址。
3. `pay_shkeeper_api_key`：服务端调用 API Key。
4. `pay_shkeeper_callback_api_key`：用于校验回调头。
5. `pay_shkeeper_default_currency`：默认 `USDTTRC20`。
6. `pay_shkeeper_order_expire_minutes`：订单过期分钟数。
7. `pay_usdt_rate_source`：汇率来源策略（后续可扩展）。

---

## 6. 接口设计（阶段一）

## 6.1 用户端

1. `POST /api/v1/user/recharge`
   - 入参：`amount`（法币金额），`pay_type`（`usdt_trc20`），`remark`
   - 出参：订单信息 + SHKeeper 支付参数（地址、币种、金额、支付链接）

2. `GET /api/v1/user/orders`
   - 保持兼容，增加 `more` 字段展示网关摘要。

## 6.2 网关回调（公开）

1. `POST /api/v1/pay/shkeeper/callback`
   - 验证头 `X-Shkeeper-Api-Key`
   - 通过 `order_id/custom` 定位内部订单
   - 仅在支付成功状态触发入账
   - 幂等：若订单已 paid，直接成功返回

## 6.3 管理端

1. `GET /api/v1/admin/orders`
2. `POST /api/v1/admin/orders/:id/mark_paid`（调试）
3. `POST /api/v1/admin/balance/adjust`（充值/扣减）
4. `GET /api/v1/admin/balance_logs`

---

## 7. 关键业务规则

1. **金额单位统一**：内部一律 `分`（int64），网关金额按字符串保留原样。
2. **幂等优先**：任何回调、手动补单都通过统一 `applyOrderPaid` 入口。
3. **事务一致性**：订单状态变更 + 余额变更 + 流水写入必须同事务。
4. **调试可回放**：管理员可手工标记支付，触发与回调一致的入账路径。
5. **不可负余额**：后台扣减走统一余额服务，低于 0 直接拒绝。

---

## 8. 安全设计

1. 回调接口只信任服务端，强校验 `X-Shkeeper-Api-Key`。
2. 记录回调原文（脱敏）到订单 `data.callback_payload`，用于审计。
3. 拒绝前端传入“已支付”状态，支付状态只能由回调/后台调试改变。
4. 建议后续增加：回调 IP 白名单 + 速率限制 + 告警（失败重试次数）。

---

## 9. 分阶段实施计划

## 阶段 A（本轮开始落地）

1. 恢复并重构 `FinanceController`。
2. 新增 SHKeeper client/service。
3. 打通 `用户充值下单 -> SHKeeper`。
4. 打通 `SHKeeper 回调 -> 自动入账`。
5. 打通 `管理员 mark_paid / balance adjust / balance logs`。
6. 前端最小改造：充值页展示地址/金额/订单号。

## 阶段 B

1. 套餐开通/续费下单统一进入订单中心。
2. 支持余额支付套餐，订单支付后自动开通/续费。
3. 自动续费 worker（余额充足自动续）。

## 阶段 C

1. 多支付通道抽象（非 SHKeeper）。
2. 汇率缓存、风控阈值、异常告警。
3. 完整 E2E 自动化测试。

---

## 10. 验收清单（阶段 A）

1. 用户创建 `usdt_trc20` 订单后可看到支付地址/金额。
2. 回调成功后：订单 `pending -> paid`，余额增加，流水新增。
3. 同一回调重复发送不重复入账。
4. 管理员手工标记已支付可触发相同入账效果。
5. 管理员加减余额必定产生日志（before/change/after/reason/source）。
6. 相关接口编译通过，核心路径可本地联调。

