package transcode

import (
	"strconv"
	"strings"
)

// ParseTimeSpec converts either a seconds string or a hh:mm:ss snippet into seconds.
func ParseTimeSpec(spec string) (float64, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, false
	}

	if strings.Contains(spec, ":") {
		parts := strings.Split(spec, ":")
		total := 0.0
		multiplier := 1.0
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			if part == "" {
				continue
			}
			val, err := strconv.ParseFloat(part, 64)
			if err != nil {
				return 0, false
			}
			total += val * multiplier
			multiplier *= 60
		}
		return total, true
	}

	val, err := strconv.ParseFloat(spec, 64)
	if err != nil || val < 0 {
		return 0, false
	}
	return val, true
}
