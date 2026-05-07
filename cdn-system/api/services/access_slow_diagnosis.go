package services

import "strings"

type DiagnoseInput struct {
	RequestTime          float64
	UpstreamConnectTime  float64
	UpstreamHeaderTime   float64
	UpstreamResponseTime float64
	UpstreamCacheStatus  string
	Status               int
	Scheme               string
	SSLProtocol          string
}

func DiagnoseAccessLogSlowReason(in DiagnoseInput) (reason string, advice string) {
	cacheStatus := normalizeCacheStatus(in.UpstreamCacheStatus)
	scheme := strings.ToLower(strings.TrimSpace(in.Scheme))

	if in.RequestTime < 1 && in.UpstreamResponseTime < 1 && in.UpstreamConnectTime < 0.3 {
		if cacheStatus == "HIT" {
			return "正常命中", "请求命中边缘缓存，耗时正常"
		}
		return "正常", "请求耗时处于正常范围"
	}
	if isMissLikeCacheStatus(cacheStatus) && in.UpstreamResponseTime >= 1 {
		return "缓存未命中回源慢", "首次访问或缓存过期正在回源；建议开启预热、延长缓存 TTL，或优化源站响应"
	}
	if isMissLikeCacheStatus(cacheStatus) {
		return "缓存未命中", "请求需要回源；热门 URL 可使用预热降低首次访问等待"
	}
	if cacheStatus == "UPDATING" {
		return "后台更新中", "边缘正在后台刷新缓存；如频繁出现可检查 TTL 是否过短或源站波动"
	}
	if cacheStatus == "STALE" {
		return "使用过期缓存兜底", "源站可能超时或返回异常，边缘已使用 stale 缓存保护用户访问"
	}
	if in.UpstreamConnectTime >= 0.5 {
		return "回源建连慢", "节点到源站 TCP/TLS 建连耗时较高；建议开启回源长连接、检查源站网络或跨境链路"
	}
	if in.UpstreamHeaderTime >= 1 {
		return "源站首包慢", "源站处理或数据库耗时较高；建议检查源站首包时间、后端接口和数据库"
	}
	if in.UpstreamResponseTime >= 1 {
		return "源站响应慢", "源站传输或大文件响应慢；建议检查源站带宽、对象大小和缓存策略"
	}
	if scheme == "https" && in.RequestTime-in.UpstreamResponseTime >= 0.5 {
		return "客户端链路或 TLS 握手慢", "总耗时明显高于回源耗时；建议检查客户端网络、TLS 会话复用和证书链"
	}
	if in.Status >= 500 {
		return "源站或节点错误", "5xx 可能来自源站错误、回源失败或节点配置异常；建议结合错误日志排查"
	}
	return "边缘处理慢", "总耗时偏高但回源耗时不高；建议检查 WAF/规则、日志采集和节点负载"
}

func normalizeCacheStatus(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isMissLikeCacheStatus(value string) bool {
	switch normalizeCacheStatus(value) {
	case "MISS", "EXPIRED", "BYPASS":
		return true
	default:
		return false
	}
}
