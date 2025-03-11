package util

import (
	"net"
)

func IsLowestIP(ipList []string, singleIP string) bool {
	parsedSingleIP := net.ParseIP(singleIP).To4()
	if parsedSingleIP == nil {
		return false
	}

	for _, ip := range ipList {
		parsedIP := net.ParseIP(ip).To4()
		if parsedIP == nil {
			continue
		}
		if parsedIP == singleIP{
			continue
		}
		for i := 0; i < 4; i++ {
			if parsedIP[i] < parsedSingleIP[i] {
				return false 
			} else if parsedIP[i] > parsedSingleIP[i] {
				break
			}
		}
	}

	return true
}