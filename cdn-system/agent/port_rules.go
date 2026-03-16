package main

import (
	"strconv"
	"strings"
)

type portRange struct {
	min int
	max int
}

func parsePortRanges(spec string) []portRange {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	parts := strings.FieldsFunc(spec, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	if len(parts) == 0 {
		return nil
	}
	ranges := make([]portRange, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if dash := strings.Index(part, "-"); dash > 0 {
			start := strings.TrimSpace(part[:dash])
			end := strings.TrimSpace(part[dash+1:])
			minPort, okMin := parsePort(start)
			maxPort, okMax := parsePort(end)
			if !okMin || !okMax {
				continue
			}
			if minPort > maxPort {
				minPort, maxPort = maxPort, minPort
			}
			ranges = append(ranges, portRange{min: minPort, max: maxPort})
			continue
		}
		port, ok := parsePort(part)
		if !ok {
			continue
		}
		ranges = append(ranges, portRange{min: port, max: port})
	}
	if len(ranges) == 0 {
		return nil
	}
	return ranges
}

func parsePort(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	port, err := strconv.Atoi(value)
	if err != nil || port <= 0 || port > 65535 {
		return 0, false
	}
	return port, true
}

func parseListenPort(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if idx := strings.LastIndex(value, "/"); idx != -1 {
		value = strings.TrimSpace(value[:idx])
	}
	if strings.HasPrefix(value, "[") {
		if idx := strings.LastIndex(value, "]"); idx != -1 && idx+1 < len(value) && value[idx+1] == ':' {
			value = value[idx+2:]
		}
	} else if idx := strings.LastIndex(value, ":"); idx != -1 {
		value = value[idx+1:]
	}
	return parsePort(value)
}

func portInRanges(port int, ranges []portRange) bool {
	for _, r := range ranges {
		if port >= r.min && port <= r.max {
			return true
		}
	}
	return false
}

func isPortAllowed(port int, allowedSpec, disabledSpec string) bool {
	allowedRanges := parsePortRanges(allowedSpec)
	disabledRanges := parsePortRanges(disabledSpec)
	if len(allowedRanges) > 0 {
		if !portInRanges(port, allowedRanges) {
			return false
		}
	}
	if len(disabledRanges) > 0 && portInRanges(port, disabledRanges) {
		return false
	}
	return true
}

func filterCustomPorts(ports []string, allowedSpec, disabledSpec string) []string {
	if len(ports) == 0 {
		return ports
	}
	if strings.TrimSpace(allowedSpec) == "" && strings.TrimSpace(disabledSpec) == "" {
		return ports
	}
	out := make([]string, 0, len(ports))
	for _, portRaw := range ports {
		port, ok := parseListenPort(portRaw)
		if !ok {
			out = append(out, portRaw)
			continue
		}
		if port == 80 || port == 443 {
			out = append(out, portRaw)
			continue
		}
		if isPortAllowed(port, allowedSpec, disabledSpec) {
			out = append(out, portRaw)
		}
	}
	return out
}
