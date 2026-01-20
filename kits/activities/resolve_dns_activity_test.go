package activities

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestResolveDNSActivity_Google(t *testing.T) {
	// save originals
	origReadHosts := readHostsFile
	origLookup := dnsLookup
	defer func() {
		readHostsFile = origReadHosts
		dnsLookup = origLookup
	}()

	// mock /etc/hosts
	readHostsFile = func(string) ([]byte, error) {
		return []byte(`
127.0.0.1 localhost
10.10.0.11 test-vpn-1-solace-a.local
`), nil
	}

	// mock DNS
	dnsLookup = func(host string) ([]net.IP, error) {
		if host == "google.com" {
			return []net.IP{
				net.ParseIP("8.8.8.8"),
				net.ParseIP("2001:4860:4860::8888"),
			}, nil
		}
		return nil, fmt.Errorf("not found")
	}

	//ip, err := ResolveDNSActivity(context.Background(), "google.com")
	//if err != nil {
	//	t.Fatalf("unexpected error: %v", err)
	//}

	//if ip != "8.8.8.8" {
	//	t.Fatalf("expected 8.8.8.8, got %s", ip)
	//}

	ip2, err2 := ResolveDNSActivity(context.Background(), "test-vpn-1-solace-a.local")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}

	if ip2 != "10.10.0.11" {
		t.Fatalf("expected 10.10.0.11, got %s", ip2)
	}
}
