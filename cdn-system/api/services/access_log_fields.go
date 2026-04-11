package services

import "fmt"

// AccessLogNormalizeHostExpr normalizes host-like values stored in access logs
// so filters and rankings treat example.com and example.com:443 as the same site.
func AccessLogNormalizeHostExpr(expr string) string {
	return fmt.Sprintf(
		"lower(replaceRegexpAll(replaceRegexpAll(trim(BOTH ' ' FROM %s), ':[0-9]+$', ''), '\\\\.$', ''))",
		expr,
	)
}

// AccessLogSiteExpr returns the ClickHouse expression used as the canonical
// site identity for access-log based statistics and listings.
func AccessLogSiteExpr() string {
	// site_name is logged as "$server_name:$server_port", so normalize it before
	// grouping/filtering. When traffic lands on the default "_" server block,
	// prefer the request Host header as fallback.
	return fmt.Sprintf(
		"if(site_name != '' AND lower(site_name) NOT LIKE '_:%%' AND lower(site_name) NOT LIKE 'localhost:%%', %s, %s)",
		AccessLogNormalizeHostExpr("site_name"),
		AccessLogNormalizeHostExpr("host"),
	)
}

// AccessLogRefererExpr groups referers by source host instead of the full URL,
// so rankings show source domains rather than being fragmented per path/query.
func AccessLogRefererExpr() string {
	return "if(http_referer = '' OR http_referer = '-', '-', lower(if(extract(http_referer, '^(?:[A-Za-z][A-Za-z0-9+.-]*://)?([^/?#]+)') != '', extract(http_referer, '^(?:[A-Za-z][A-Za-z0-9+.-]*://)?([^/?#]+)'), http_referer)))"
}

// AccessLogRealSiteTrafficCondition filters out internal/default host noise so
// rankings reflect actual matched site traffic instead of raw Host header junk.
func AccessLogRealSiteTrafficCondition() string {
	siteExpr := AccessLogSiteExpr()
	return fmt.Sprintf(
		"(%s != '' AND %s != '_' AND %s NOT IN ('127.0.0.1', 'localhost') AND NOT match(%s, '^[0-9]{1,3}(\\\\.[0-9]{1,3}){3}$') AND NOT match(%s, '^[0-9a-f:]+$'))",
		siteExpr,
		siteExpr,
		siteExpr,
		siteExpr,
		siteExpr,
	)
}
