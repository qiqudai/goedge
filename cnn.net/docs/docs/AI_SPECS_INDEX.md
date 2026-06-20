# AI 任务文档总索引（cnn.net）

## 1. 核心数据面
1. `YARP_DYNAMIC_ROUTING_HOT_RELOAD_AI_SPEC.md`
2. `ACL_WAF_CC_RULE_ENGINE_AI_SPEC.md`
3. `TLS_CERT_SNI_HOT_RELOAD_AI_SPEC.md`
4. `STREAM_L4_PROXY_AI_SPEC.md`

## 2. 扩展与治理
1. `PLUGIN_DLL_RUNTIME_AI_SPEC.md`
2. `DEBUG_LOG_SWITCH_AUDIT_AI_SPEC.md`
3. `CONTROL_PLANE_SYNC_TASK_AI_SPEC.md`
4. `PERF_CAPACITY_STABILITY_AI_SPEC.md`
5. `PERMISSION_HIERARCHY_FRAMEWORK_AI_SPEC.md`
6. `LOGGING_RW_PIPELINE_AI_SPEC.md`
7. `AGENT_TASK_PAYLOAD_CONTRACTS.md`
8. `AI_IMPLEMENTATION_ROADMAP_MASTER.md`
9. `DOCUMENT_DELIVERY_CHECKLIST.md`
10. `FEATURE_LIMITS_AND_TACTICS_MATRIX.md`
11. `CK_WS_SYNC_E2E_TEST_AI_SPEC.md`

## 3. 总体设计参考
1. `../CNN_NET_REMAINING_DESIGN_FULL.md`
2. `YARP_DYNAMIC_ROUTING_HOT_RELOAD_DESIGN.md`（面向人）

## 4. 建议执行顺序（给 AI）
1. YARP 动态路由
2. TLS/SNI
3. ACL/WAF/CC
4. 控制面同步与任务幂等
5. 调试开关与审计
6. 插件体系
7. L4 Stream
8. 性能与稳定性门禁

## 5. 完成标准
- 每个 AI_SPEC 的 Definition of Done 全部满足。
- 每个 AI_SPEC 的测试与验收命令全部通过。

## 6. 文档完备性状态（2026-04-01）
- 总体设计：已覆盖
- 框架设计：已覆盖
- 权限分级：已覆盖
- 日志读写与调试开关：已覆盖
- 任务载荷契约：已覆盖
- 性能/容量/稳定性：已覆盖
- AI 开发路线图：已覆盖
