package services

// AccessLogSiteExpr returns the ClickHouse expression used as the canonical
// site identity for access-log based statistics and listings.
func AccessLogSiteExpr() string {
	return "if(site_name != '', site_name, host)"
}

// AccessLogRealSiteTrafficCondition filters out internal/default host noise so
// rankings reflect actual matched site traffic instead of raw Host header junk.
func AccessLogRealSiteTrafficCondition() string {
	return "((site_name != '' AND site_name NOT LIKE 'localhost:%' AND site_name NOT LIKE '_:%') OR (site_name = '' AND host != '' AND host != '_' AND host NOT LIKE '%.%.%.%'))"
}
