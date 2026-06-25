package ip

import (
	"net"
	"net/http"
)

func CheckIP(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// При запросе эндпоинта /api/internal/stats нужно проверять,
			// что переданный в заголовке запроса X-Real-IP IP-адрес клиента входит в доверенную подсеть,
			// в противном случае возвращать статус ответа 403 Forbidden.
			// При пустом значении переменной trusted_subnet доступ к эндпоинту должен быть запрещён для любого входящего запроса.

			if trustedSubnet == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ip := r.Header.Get("X-Real-IP")
			if ip == "" || !isIPInTrustedSubnet(ip, trustedSubnet) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Функция для проверки, входит ли IP-адрес в доверенную подсеть
func isIPInTrustedSubnet(ip, trustedSubnet string) bool {
	// Проверяем, что IP-адрес и доверенная подсеть корректны
	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	// Получаем объект ipNet из доверенной подсети
	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return false
	}

	// Проверяем, входит ли IP-адрес в доверенную подсеть
	return subnet.Contains(clientIP)
}
