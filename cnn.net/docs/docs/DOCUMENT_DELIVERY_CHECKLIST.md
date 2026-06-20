# 文档交付清单（最终）

## 1. 目标
- 本清单用于确认 `cnn.net` 设计文档已达到可直接驱动 AI 开发的完备状态。

## 2. 文档清单与用途
1. `../CNN_NET_REMAINING_DESIGN_FULL.md`
- 用途：全局架构、功能范围、峰值与稳定性目标。

2. `CODE_FRAMEWORK_DESIGN_V1.md`
- 用途：代码分层、低耦合与高复用框架约束。

3. `../AI_DEVELOPMENT_RULES_V2.md`
- 用途：AI 编码统一规范与不可违反约束。

4. `YARP_DYNAMIC_ROUTING_HOT_RELOAD_AI_SPEC.md`
- 用途：动态路由与热更新实施细节。

5. `ACL_WAF_CC_RULE_ENGINE_AI_SPEC.md`
- 用途：安全规则引擎实现。

6. `TLS_CERT_SNI_HOT_RELOAD_AI_SPEC.md`
- 用途：TLS/SNI 热更新策略。

7. `PLUGIN_DLL_RUNTIME_AI_SPEC.md`
- 用途：动态 DLL 插件运行时。

8. `DEBUG_LOG_SWITCH_AUDIT_AI_SPEC.md`
- 用途：调试开关、审计、人工调试日志。

9. `CONTROL_PLANE_SYNC_TASK_AI_SPEC.md`
- 用途：控制面配置同步与任务状态机。

10. `STREAM_L4_PROXY_AI_SPEC.md`
- 用途：L4 流量代理实现。

11. `PERF_CAPACITY_STABILITY_AI_SPEC.md`
- 用途：性能、容量、稳定性门禁。

12. `PERMISSION_HIERARCHY_FRAMEWORK_AI_SPEC.md`
- 用途：权限分级与资源域授权。

13. `LOGGING_RW_PIPELINE_AI_SPEC.md`
- 用途：统一日志读写框架。

14. `AGENT_TASK_PAYLOAD_CONTRACTS.md`
- 用途：任务载荷与 ack 协议契约。

15. `AI_IMPLEMENTATION_ROADMAP_MASTER.md`
- 用途：阶段路线图、测试门禁、上线判定。

16. `AI_SPECS_INDEX.md`
- 用途：总索引与完备性状态。

## 3. 可执行性检查
- 每份文档均包含：
  - 明确目标
  - 边界与限制
  - 关键实现点
  - 性能/稳定性要求
  - 测试与 DoD
- 协议类文档包含 JSON 示例，避免歧义。
- 框架类文档包含接口草案，便于 AI 直接落代码。

## 4. 结论
- 文档体系已可进入“按规范逐模块编码与验收”阶段。
