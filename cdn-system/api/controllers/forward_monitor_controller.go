package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"net/http"
	"sort"
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
	allowedPorts := []int{}

	if isUserRequest(ctx) {
		userID := parseInt64(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}}})
			return
		}
		portMap, ports, err := loadUserForwardPortMap(userID)
		if err != nil || len(ports) == 0 {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}}})
			return
		}
		if port > 0 {
			if !portAllowed(portMap, port, protocol) {
				ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}}})
				return
			}
			allowedPorts = []int{port}
		} else if protocol != "" {
			filtered := filterPortsByProtocol(portMap, protocol)
			if len(filtered) == 0 {
				ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"x_axis": []string{}, "bandwidth": []float64{}, "traffic": []float64{}}})
				return
			}
			allowedPorts = filtered
		} else {
			allowedPorts = ports
		}
	}

	buckets, err := services.QueryForwardTrafficBucketsWithPorts(start, end, bucketMinutes, port, protocol, allowedPorts)
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
	allowedPorts := []int{}
	if isUserRequest(ctx) {
		userID := parseInt64(mustGet(ctx, "userID"))
		if userID == 0 {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []forwardRankingItem{}}})
			return
		}
		_, ports, err := loadUserForwardPortMap(userID)
		if err != nil || len(ports) == 0 {
			ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []forwardRankingItem{}}})
			return
		}
		allowedPorts = ports
	}
	list, err := services.QueryForwardPortRankingWithPorts(start, end, 50, allowedPorts)
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

func loadUserForwardPortMap(userID int64) (map[int]map[string]bool, []int, error) {
	var forwards []models.Forward
	if err := db.DB.Select("listen").Where("uid = ?", userID).Find(&forwards).Error; err != nil {
		return nil, nil, err
	}
	portMap := map[int]map[string]bool{}
	for _, forward := range forwards {
		for _, entry := range forward.ListenPorts {
			port, protocol := parseForwardListenPort(entry)
			if port <= 0 {
				continue
			}
			if portMap[port] == nil {
				portMap[port] = map[string]bool{}
			}
			if protocol == "" {
				portMap[port]["TCP"] = true
			} else {
				portMap[port][protocol] = true
			}
		}
	}
	ports := make([]int, 0, len(portMap))
	for port := range portMap {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return portMap, ports, nil
}

func parseForwardListenPort(raw string) (int, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ""
	}
	protocol := ""
	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		raw = strings.TrimSpace(parts[0])
		protocol = strings.ToUpper(strings.TrimSpace(parts[1]))
	}
	if protocol != "TCP" && protocol != "UDP" {
		protocol = ""
	}
	portStr := ""
	for _, r := range raw {
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

func portAllowed(portMap map[int]map[string]bool, port int, protocol string) bool {
	entry, ok := portMap[port]
	if !ok {
		return false
	}
	if protocol == "" {
		return true
	}
	return entry[protocol]
}

func filterPortsByProtocol(portMap map[int]map[string]bool, protocol string) []int {
	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	out := []int{}
	for port, protocols := range portMap {
		if protocols[protocol] {
			out = append(out, port)
		}
	}
	sort.Ints(out)
	return out
}
