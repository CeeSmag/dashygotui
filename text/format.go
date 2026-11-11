package text

import (
	"fmt"
	"math"
)

// Bytes formats a size for display, counting in 1024s: "412.3 GB", "0 B".
// Negative sizes are shown as zero.
func Bytes(n int64) string {
	if n < 0 {
		n = 0
	}
	v := float64(n)
	units := [...]string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}

// Rate formats a transfer rate in bytes per second, counting in 1000s:
// "1.2 MB/s". NaN and negative rates are shown as zero.
func Rate(bps float64) string {
	bps = cleanRate(bps)
	units := [...]string{"B/s", "KB/s", "MB/s", "GB/s", "TB/s"}
	i := 0
	for bps >= 1000 && i < len(units)-1 {
		bps /= 1000
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", bps, units[i])
	}
	return fmt.Sprintf("%.1f %s", bps, units[i])
}

// RateShort formats a rate compactly, for graph scales: "5M/s".
func RateShort(bps float64) string {
	bps = cleanRate(bps)
	units := [...]string{"", "K", "M", "G", "T"}
	i := 0
	for bps >= 1000 && i < len(units)-1 {
		bps /= 1000
		i++
	}
	if bps >= 10 || bps == math.Trunc(bps) {
		return fmt.Sprintf("%.0f%s/s", bps, units[i])
	}
	return fmt.Sprintf("%.1f%s/s", bps, units[i])
}

func cleanRate(bps float64) float64 {
	if math.IsNaN(bps) || bps < 0 {
		return 0
	}
	return bps
}
