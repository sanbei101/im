package mq

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"strings"
)

// machineFingerprint is a stable per-host identifier used to derive consumer
// names. It is computed once at package init and never changes for the
// lifetime of the process.
var machineFingerprint = generateMachineFingerprint(16)

// generateMachineFingerprint builds a stable identifier from the host's
// machine-id, primary MAC, and hostname. The result is hashed with SHA-256
// and truncated to `truncateLen` hex characters so consumer names stay
// short. Docker / virtual interfaces are filtered out so the fingerprint
// is stable across Docker bridge setups.
func generateMachineFingerprint(truncateLen int) string {
	if truncateLen < 8 || truncateLen > 64 {
		truncateLen = 16
	}

	var parts []string

	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id := strings.TrimSpace(string(data))
			if len(id) >= 32 {
				parts = append(parts, id[:32])
			}
		}
	}

	if mac := primaryMAC(); mac != "" {
		parts = append(parts, mac)
	}

	if len(parts) == 0 {
		host, err := os.Hostname()
		if err == nil && host != "" {
			parts = append(parts, host)
		} else {
			parts = append(parts, "unknown")
		}
	}

	raw := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(raw))
	fullHex := hex.EncodeToString(hash[:])

	if truncateLen > len(fullHex) {
		truncateLen = len(fullHex)
	}
	return fullHex[:truncateLen]
}

// primaryMAC returns the first non-virtual, non-loopback MAC address as a
// hex string with colons stripped. Empty string if none is found.
func primaryMAC() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if strings.Contains(iface.Name, "docker") ||
			strings.Contains(iface.Name, "veth") ||
			strings.Contains(iface.Name, "br-") {
			continue
		}
		mac := iface.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" {
			return strings.ReplaceAll(mac, ":", "")
		}
	}
	return ""
}
