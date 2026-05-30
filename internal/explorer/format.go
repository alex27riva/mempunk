package explorer

import (
	"fmt"
	"math"
	"time"
)

// BTCToSats converts a BTC float64 (as returned by RPC) to satoshis.
func BTCToSats(btc float64) int64 {
	return int64(math.Round(btc * 1e8))
}

// FormatBTC formats satoshis as a BTC string with 8 decimal places.
func FormatBTC(sats int64) string {
	neg := sats < 0
	if neg {
		sats = -sats
	}
	s := fmt.Sprintf("%d.%08d", sats/1e8, sats%1e8)
	if neg {
		return "-" + s
	}
	return s
}

// FormatAge returns a human-readable duration since unixTime.
func FormatAge(unixTime int64) string {
	d := time.Since(time.Unix(unixTime, 0))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(bytes int) string {
	switch {
	case bytes >= 1_000_000:
		return fmt.Sprintf("%.2f MB", float64(bytes)/1_000_000)
	case bytes >= 1_000:
		return fmt.Sprintf("%.1f kB", float64(bytes)/1_000)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSizeI64 is FormatSize for int64 values (e.g. size_on_disk).
func FormatSizeI64(bytes int64) string { return FormatSize(int(bytes)) }

// ShortenHash shortens a 64-char hash to "abcd1234…ef567890".
func ShortenHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:8] + "…" + hash[len(hash)-8:]
}

// FormatVersion converts a Bitcoin Core CLIENT_VERSION integer to "M.m.p".
// e.g. 270100 → "27.1.0"
func FormatVersion(v int) string {
	return fmt.Sprintf("%d.%d.%d", v/10000, (v%10000)/100, v%100)
}
