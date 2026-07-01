package ip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тесты для функции isIPInTrustedSubnet
func Test_isIPInTrustedSubnet(t *testing.T) {
	tests := []struct {
		name          string
		ip            string
		trustedSubnet string
		want          bool
	}{
		{
			name:          "IP входит в подсеть",
			ip:            "192.168.1.100",
			trustedSubnet: "192.168.1.0/24",
			want:          true,
		},
		{
			name:          "IP не входит в подсеть",
			ip:            "10.0.0.1",
			trustedSubnet: "192.168.1.0/24",
			want:          false,
		},
		{
			name:          "некорректный IP",
			ip:            "не-ip",
			trustedSubnet: "192.168.1.0/24",
			want:          false,
		},
		{
			name:          "некорректная подсеть",
			ip:            "192.168.1.100",
			trustedSubnet: "не-подсеть",
			want:          false,
		},
		{
			name:          "граничный IP - первый адрес сети",
			ip:            "192.168.1.0",
			trustedSubnet: "192.168.1.0/24",
			want:          true,
		},
		{
			name:          "граничный IP - последний адрес сети",
			ip:            "192.168.1.255",
			trustedSubnet: "192.168.1.0/24",
			want:          true,
		},
		{
			name:          "IPv6 входит в подсеть",
			ip:            "2001:db8::1",
			trustedSubnet: "2001:db8::/32",
			want:          true,
		},
		{
			name:          "IPv6 не входит в подсеть",
			ip:            "2001:db9::1",
			trustedSubnet: "2001:db8::/32",
			want:          false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isIPInTrustedSubnet(test.ip, test.trustedSubnet)
			assert.Equal(t, test.want, got)
		})
	}
}
