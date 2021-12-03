package script

import (
	"net"
)

func GetIpAddress() string {
	addrs, _ := net.InterfaceAddrs()
	for _, value := range addrs {
		if ipnet, ok := value.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
