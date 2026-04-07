package services

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func RoundFloat(val float64, precision int) float64 {
	if precision < 0 {
		return val
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(val*factor) / factor
}

func BytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func BytesToMbps(bytes uint64, bucket time.Duration) float64 {
	seconds := bucket.Seconds()
	if seconds <= 0 {
		return 0
	}
	return float64(bytes) * 8 / seconds / 1_000_000
}

func FormatBytes(bytes uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	val := float64(bytes)
	idx := 0
	for val >= 1024 && idx < len(units)-1 {
		val /= 1024
		idx++
	}
	if idx == 0 {
		return strconv.FormatUint(bytes, 10) + " B"
	}
	return fmt.Sprintf("%.2f %s", val, units[idx])
}

func FormatBandwidth(mbps float64) string {
	if mbps <= 0 {
		return "0 Mbps"
	}
	if mbps >= 1000 {
		return fmt.Sprintf("%.2f Gbps", mbps/1000)
	}
	if mbps < 1 {
		return fmt.Sprintf("%.2f Kbps", mbps*1000)
	}
	return fmt.Sprintf("%.2f Mbps", mbps)
}

func FormatCount(count uint64) string {
	return strconv.FormatUint(count, 10)
}
