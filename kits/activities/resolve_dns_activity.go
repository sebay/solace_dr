package activities

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
)

var readHostsFile = os.ReadFile
var dnsLookup = net.LookupIP

func ResolveDNSActivity(ctx context.Context, dnsName string) (string, error) {
	// todo: this is a hack just for quick test....
	// 1. Try /etc/hosts first
	hosts, err := os.ReadFile("/etc/hosts")
	if err == nil {
		lines := strings.Split(string(hosts), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			ip := fields[0]
			for _, name := range fields[1:] {
				if name == dnsName {
					return ip, nil
				}
			}
		}
	}

	// 2. Fallback to normal DNS lookup
	ips, err := net.LookupIP(dnsName)
	if err != nil || len(ips) == 0 {
		return "", err
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 found for %s", dnsName)
}
