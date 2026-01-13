package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cdn-api/services"

	"github.com/gin-gonic/gin"
)

type ForwardMonitorController struct{}

type forwardRankingItem struct {
	Port        string `json:"port"`
	Connections uint64 `json:"connections"`
	Traffic     string `json:"traffic"`
}

func (c *ForwardMonitorController) Traffic(ctx *gin.Context) {
	rangeKey := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("range", "1h")))
	keyword := strings.TrimSpace(ctx.Query("keyword"))
	start, end, step, bucketMinutes, labelFormat := resolveForwardRange(rangeKey)
	port, protocol := parseForwardKeyword(keyword)

	buckets, err := services.QueryForwardTrafficBuckets(start, end, bucketMinutes, port, protocol)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}}})
		return
	}
	factor := resolveForwardTrafficFactor()

	bucketMap := make(map[time.Time]uint64, len(buckets))
	for _, bucket := range buckets {
		key := bucket.Bucket.Truncate(step)
		bucketMap[key] = bucket.TotalBytes
	}

	times := make([]string, 0)
	bandwidth := make([]float64, 0)
	traffic := make([]float64, 0)

	cur := start
	for !cur.After(end) {
		totalBytes := bucketMap[cur]
		if factor != 1 {
			totalBytes = uint64(float64(totalBytes) * factor)
		}
		trafficGB := float64(totalBytes) / (1024 * 1024 * 1024)
		bandwidthMbps := 0.0
		if step.Seconds() > 0 {
			bandwidthMbps = float64(totalBytes) * 8 / (step.Seconds() * 1000 * 1000)
		}
		times = append(times, cur.Format(labelFormat))
		bandwidth = append(bandwidth, services.RoundFloat(bandwidthMbps, 3))
		traffic = append(traffic, services.RoundFloat(trafficGB, 3))
		cur = cur.Add(step)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"x_axis":    times,
			"bandwidth": bandwidth,
			"traffic":   traffic,
		},
	})
}

func (c *ForwardMonitorController) Ranking(ctx *gin.Context) {
	rangeKey := strings.ToLower(strings.TrimSpace(ctx.DefaultQuery("range", "1h")))
	start, end, _, _, _ := resolveForwardRange(rangeKey)
	list, err := services.QueryForwardPortRanking(start, end, 50)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []forwardRankingItem{}}})
		return
	}
	factor := resolveForwardTrafficFactor()

	resp := make([]forwardRankingItem, 0, len(list))
	for _, item := range list {
		adjusted := float64(item.TotalBytes)
		if factor != 1 {
			adjusted = adjusted * factor
		}
		trafficGB := adjusted / (1024 * 1024 * 1024)
		portLabel := fmt.Sprintf("%d/%s", item.Port, strings.ToUpper(strings.TrimSpace(item.Protocol)))
		if strings.TrimSpace(item.Protocol) == "" {
			portLabel = fmt.Sprintf("%d/TCP", item.Port)
		}
		resp = append(resp, forwardRankingItem{
			Port:        portLabel,
			Connections: item.Connections,
			Traffic:     fmt.Sprintf("%.2f GB", trafficGB),
		})
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list": resp,
		},
	})
}

func resolveForwardRange(rangeKey string) (time.Time, time.Time, time.Duration, int, string) {
	end := time.Now()
	switch rangeKey {
	case "6h":
		step := 5 * time.Minute
		start := end.Add(-6 * time.Hour).Truncate(step)
		return start, end.Truncate(step), step, 5, "15:04"
	case "24h":
		step := 30 * time.Minute
		start := end.Add(-24 * time.Hour).Truncate(step)
		return start, end.Truncate(step), step, 30, "01-02 15:04"
	default:
		step := time.Minute
		start := end.Add(-1 * time.Hour).Truncate(step)
		return start, end.Truncate(step), step, 1, "15:04"
	}
}

func parseForwardKeyword(keyword string) (int, string) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return 0, ""
	}
	portPart := keyword
	protocol := ""
	if strings.Contains(keyword, "/") {
		parts := strings.SplitN(keyword, "/", 2)
		portPart = strings.TrimSpace(parts[0])
		protocol = strings.TrimSpace(parts[1])
	}
	lower := strings.ToLower(keyword)
	if protocol == "" {
		if strings.Contains(lower, "tcp") {
			protocol = "TCP"
		} else if strings.Contains(lower, "udp") {
			protocol = "UDP"
		}
	}
	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	if protocol != "TCP" && protocol != "UDP" {
		protocol = ""
	}
	portStr := ""
	for _, r := range portPart {
		if r >= '0' && r <= '9' {
			portStr += string(r)
		}
	}
	if portStr == "" {
		return 0, protocol
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return 0, protocol
	}
	return port, protocol
}

func resolveForwardTrafficFactor() float64 {
	cfg, err := services.LoadConfigMap("system", "global", 0)
	if err != nil {
		return 1.0
	}
	raw := strings.TrimSpace(cfg["tcp_traffic_factor"])
	if raw == "" {
		return 1.0
	}
	factor, err := strconv.ParseFloat(raw, 64)
	if err != nil || factor <= 0 {
		return 1.0
	}
	return factor
}
