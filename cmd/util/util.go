package util

import (
	"net"
	"time"
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
		
		// Skip if it's the same IP
		if ip == singleIP {
			continue
		}
		
		// Compare byte by byte
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



func IsMaster(ipMap map[string]time.Time, singleIP string) bool {

	ipList := make([]string, 0, len(ipMap))
	for nodeID := range ipMap {
		ipList = append(ipList, nodeID)
	}

	return IsLowestIP(ipList, singleIP)
}
