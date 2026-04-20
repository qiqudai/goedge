# Error Codes

统一返回结构：`{"code":200,"message":"Success","data":...}`，message 本地化。

| code | http | message | 说明 | 典型场景 |
|---|---|---|---|---|
| 200 | 200 | Success | 成功 | - |
| 40001 | 400 | BadRequest | 参数非法/缺失 | 域名格式/端口范围错误 |
| 40101 | 401 | Unauthorized | token 无效/缺失 | 登录态失效 |
| 40301 | 403 | Forbidden | 权限不足 | 非管理员访问 admin 接口 |
| 40401 | 404 | NotFound | 资源不存在 | site/node/cert 不存在 |
| 40901 | 409 | Conflict | 资源冲突 | 重复域名/并发修改 |
| 42901 | 429 | RateLimited | 请求过多 | 登录/任务频率限制 |
| 50001 | 500 | InternalError | 未分类内部错误 | 兜底 |
| 50201 | 502 | BadGateway | 外部系统错误 | DNS/ACME 失败 |
| 50301 | 503 | ServiceUnavailable | 系统不可用 | 队列拥塞/依赖异常 |
