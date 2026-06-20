# 分页开发规范（TablePager）

## 1. 目标
- 全站分页行为一致：搜索、刷新、切页、改 pageSize、开关分页都由 `TablePager` 统一驱动。
- 页面只负责业务过滤参数与列表渲染，不手写重复分页状态机。

## 2. 接入模板
- 页面中放置 `TablePager`：
  - 必填：`StateKey`、`Total`、`QueryChanged`
  - 建议：`DefaultPageSize="20"`、`UnpagedLimit="1000"`
- 在 `@code` 中维护 `TablePageQuery _pageQuery` 快照（用于诊断与调试）。
- 在 `OnPageQueryChangedAsync(TablePageQuery query)` 中：
  - 先保存 `_pageQuery = query`
  - 再把 `query.Page / query.PageSize` 映射到请求 DTO
  - 最后调用 `LoadAsync()`

## 3. 状态 key 命名
- 统一格式：`{domain}:{page}:{table}` 或 `{domain}:{page}:{subview}:table`
- 示例：
  - `website:list:table`
  - `website:access_logs:history:table`
  - `node:list:monitor:table`

## 4. 搜索/刷新语义
- 关键词检索按钮统一文案为“搜索”。
- 主动拉新按钮统一文案为“刷新”。
- 行为规则：
  - `搜索`：调用 `TablePager.SearchAsync()`，回到第 1 页。
  - `刷新`：调用 `TablePager.RefreshAsync()`，保持当前页。
- 不要在分页页里让按钮直接调用 `Load*` 作为常规入口。

## 5. 后端契约
- 支持服务端分页的列表接口统一接收：`page`、`pageSize`
- 统一返回：`total`、`list`
- 关闭分页时允许整表查询，但上限不超过 `UnpagedLimit`（默认 1000）。

## 6. 并发与竞态
- 快速切页/改 pageSize/开关分页时，依赖 `TablePager` 的并发合并保护。
- 页面层不要自行叠加二次防抖或重入控制导致行为冲突；仅保留必要的 `_loading` 保护。

## 7. 可观测性
- 建议在关键分页页保留可开关“分页诊断”面板：
  - `stateKey`
  - `pagingEnabled`
  - `page/pageSize`
  - `loaded/total`
- `TablePager` 统一记录结构化日志字段：`stateKey`、`pagingEnabled`、`page`、`pageSize`、`total`。
- 统一检索/仪表盘建议按 `uiEvent=table_pager_query` 聚合，并按以下维度切片：
  - `stateKey`
  - `page`
  - `pageSize`
  - `pagingEnabled`
  - `total`
- 推荐快速检索关键字：`uiEvent=table_pager_query stateKey=<your_state_key>`

### 7.1 统一检索/仪表盘实际入口（2026-04-20）
- Web 入口：`/dashboard`
  - 页面文件：`src/Cnn.Api/Pages/Dashboard.razor`
  - 菜单入口：`src/Cnn.Api/Shared/NavMenu.razor`（“仪表盘”）
- 后端聚合接口：
  - 管理端：`GET /api/v1/admin/dashboard`
  - 用户端：`GET /api/v1/user/dashboard`
- 分页日志产出点：
  - `src/Cnn.Api/Shared/TablePager.razor`（`uiEvent=table_pager_query` 结构化日志）
- 排查时统一动作：
  1. 在业务页面打开“分页诊断”获取 `stateKey/page/pageSize`。
  2. 在统一日志检索中按 `uiEvent=table_pager_query stateKey=<stateKey>` 过滤。
  3. 结合 `/dashboard` 指标面板做时间窗口对齐。

## 8. 最小代码清单
- 组件：`TablePager.razor`
- 查询模型：`TablePageQuery.cs`
- 页面必备方法：
  - `OnPageQueryChangedAsync(TablePageQuery query)`
  - `SearchAsync()` -> `TablePager.SearchAsync()`
  - `RefreshAsync()` -> `TablePager.RefreshAsync()`

## 9. 验收清单
- 首次加载：先恢复历史 `pageSize`，再发起首个请求。
- 切页：请求参数与 UI 页码一致。
- 改 pageSize：请求参数变化正确。
- 关闭分页：整表查询且不超过上限。
- 快速操作：仅落最新状态，无请求风暴。

## 10. 常见坑
- 搜索入口直接调用 `LoadAsync()`：
  - 现象：搜索后仍停留在旧页码。
  - 修复：统一调用 `TablePager.SearchAsync()`；无 `TablePager` 引用时，将 `_pageQuery` 重建为 `Page = 1` 后再加载。
- 刷新入口直接把页码重置为 1：
  - 现象：点击“刷新”后丢失当前页上下文。
  - 修复：统一调用 `TablePager.RefreshAsync()`；无 `TablePager` 引用时仅触发加载，不改页码。
- 页面内手写 `_page/_pageSize` 与 `_pageQuery` 双状态：
  - 现象：诊断面板显示页码与实际请求参数不一致。
  - 修复：请求参数只读 `_pageQuery.Page/_pageQuery.PageSize`；移除重复状态。
- 首屏请求早于状态恢复：
  - 现象：第一次加载总是默认 `pageSize=20`，刷新后才生效历史配置。
  - 修复：让 `TablePager` 触发首请，不在 `OnInitializedAsync` 里手动调用首个 `LoadAsync`。
- 快速操作下请求风暴：
  - 现象：短时间内堆积多个旧请求，列表回跳。
  - 修复：只通过 `TablePager` 发起分页动作，避免页面层再叠加竞争性的防抖/轮询逻辑。

## 11. 排查步骤
1. 打开页面“分页诊断”并记录 `stateKey/pagingEnabled/page/pageSize/loaded/total`。
2. 在日志中检索 `uiEvent=table_pager_query`，按同一 `stateKey` 过滤，确认参数轨迹。
3. 验证语义：
   - 点击“搜索”后首条请求应为 `page=1`。
   - 点击“刷新”后请求应保持当前 `page`。
4. 快速连续操作（切页、改 pageSize、搜索）后，确认最终渲染与最后一次操作一致。
5. 若参数与 UI 不一致，优先检查：
   - 是否仍存在 `_page/_pageSize` 重复状态；
   - 是否有按钮仍直连 `Load*`；
   - 是否在 `OnInitializedAsync` 中抢先触发了列表请求。

## 12. 未纳入分页规范的非分页页清单
- `src/Cnn.Api/Pages/Forward/Monitor.razor`（无 `TablePager`）
- `src/Cnn.Api/Pages/Website/Statistics.razor`（无 `TablePager`）
- `src/Cnn.Api/Pages/System/Upgrade.razor`（无 `TablePager`）
- 维护规则：
  - 新页面若不包含 `<TablePager`，默认不纳入本规范验收矩阵。
  - 只有引入 `<TablePager` 后，才需要补“搜索/刷新语义 + 页面级回归”。
