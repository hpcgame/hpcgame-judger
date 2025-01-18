package framework

import (
	"net"
)

func GetIPAddress() string {
	ifs := Must(net.Interfaces())
	for _, i := range ifs {
		addrs := Must(i.Addrs())
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String()
				}
			}
		}
	}
	PanicString("No IP address found")
	return ""
}
