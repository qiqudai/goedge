# AI 实施规范：性能、容量与长期稳定性（无歧义版）

> 目标：给 AI 明确的“必须实现的性能治理点”，避免只完成功能不做稳定性工程。

---

## 0. 执行边界

### 0.1 允许修改
- `src/Cnn.Agent/*`
- `src/Cnn.Api/*`（仅新增观测接口或指标，不改业务语义）
- `docs/tests/*`（新增压测脚本与基线）

### 0.2 禁止事项
- `MUST NOT` 牺牲正确性换性能。
- `MUST NOT` 跳过观测直接上优化。

---

## 1. Definition of Done

1. 核心链路有指标：QPS、延迟、错误率、配置应用耗时。
2. 有容量基线：10k/50k/100k RPS 测试结果。
3. 有稳定性基线：72h soak test 结果。
4. 有性能回归门禁脚本。
5. 有 GC/内存分配观测与阈值告警。

---

## 2. 必须新增内容

### 2.1 指标
- 请求：`requests_total`, `request_duration_ms`
- 配置：`proxy_apply_duration_ms`, `proxy_apply_fail_total`
- 安全：`security_rule_hits_total`
- 缓存：`cache_hit_ratio`
- 资源：`process_cpu`, `process_working_set`, `gc_gen2_collections`

### 2.2 压测脚本（必须）
- `docs/tests/perf/l7_rps_10k.sh`
- `docs/tests/perf/l7_rps_50k.sh`
- `docs/tests/perf/l7_rps_100k.sh`
- `docs/tests/perf/hot_reload_during_traffic.sh`
- `docs/tests/perf/soak_72h.sh`

### 2.3 门禁脚本（必须）
- `docs/tests/perf/perf_gate.sh`
- 规则：
  - P99 不得劣化超过基线 10%
  - 错误率不得超过基线 0.05%
  - 配置热更期间无显著错误尖峰

---

## 3. 代码级性能约束（MUST）

1. 请求路径禁止 JSON 反序列化。
2. 请求路径禁止大对象分配（> 85KB）。
3. 热路径字符串处理尽量使用缓存/Span。
4. Regex 必须预编译并可控。
5. 高频字典查询必须使用不区分大小写固定 comparer。

---

## 4. 稳定性约束（MUST）

1. 所有后台循环必须支持取消令牌。
2. 重试必须带退避与上限。
3. 关键写盘操作必须原子。
4. 所有失败路径必须记录可追踪日志（含 trace_id/version/task_id）。

---

## 5. 必须测试清单

1. Perf_Baseline_10kRps
2. Perf_Baseline_50kRps
3. Perf_Baseline_100kRps
4. Perf_HotReload_WithTraffic
5. Stability_Soak72h
6. Memory_NoUnboundedGrowth
7. Gc_Gen2_WithinThreshold
8. ErrorRate_WithinThreshold

---

## 6. 验收命令

```bash
rg -n "requests_total|request_duration|proxy_apply_duration|gc_gen2" src
```

```bash
ls -la docs/tests/perf
```

---

## 7. 交付格式

AI 输出必须包含：
1. 指标清单与埋点位置
2. 压测结果表格
3. 与基线对比结论
4. 是否通过 perf_gate
