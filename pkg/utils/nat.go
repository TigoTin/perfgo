package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ccding/go-stun/stun"
)

var (
	natCache     = make(map[string]*natCacheEntry)
	natCacheLock sync.RWMutex
	cacheExpiry  = 10 * time.Minute
)

type natCacheEntry struct {
	natType     string
	publicIP    string
	expiresAt   time.Time
	lastChecked time.Time
}

var stunServers = []string{
	"stun.qcloud.com:3478",
	"stun.miwifi.com:3478",
	"stun.ekiga.net:3478",
	"stun.ideasip.com:3478",
	"stun.voiparound.com:3478",
	"stun.l.google.com:19302",
	"stun.aliyun.com:3478",
}

var natTypeMap = map[stun.NATType]string{
	stun.NATNone:                  "NAT0",
	stun.NATFull:                  "NAT1",
	stun.NATRestricted:            "NAT2",
	stun.NATPortRestricted:        "NAT3",
	stun.NATSymetric:              "NAT4",
	stun.NATBlocked:               "NAT5",
	stun.SymmetricUDPFirewall:     "NAT6",
}

func getNATCache(localIP string) (*natCacheEntry, bool) {
	natCacheLock.RLock()
	defer natCacheLock.RUnlock()

	entry, exists := natCache[localIP]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry, true
}

func setNATCache(localIP, natType, publicIP string) {
	natCacheLock.Lock()
	defer natCacheLock.Unlock()

	natCache[localIP] = &natCacheEntry{
		natType:     natType,
		publicIP:    publicIP,
		expiresAt:   time.Now().Add(cacheExpiry),
		lastChecked: time.Now(),
	}
}

func DetectNATType(localIP string) (string, string, error) {
	if entry, exists := getNATCache(localIP); exists {
		return entry.natType, entry.publicIP, nil
	}

	natType, publicIP, err := detectNATTypeWithRetry(localIP)
	if err == nil {
		setNATCache(localIP, natType, publicIP)
	}

	return natType, publicIP, err
}

func detectNATTypeWithRetry(localIP string) (string, string, error) {
	for _, serverAddr := range stunServers {
		nat, publicIP, err := detectWithSTUNServer(serverAddr, localIP)
		if err == nil && nat != "Unknown" {
			return nat, publicIP, nil
		}
	}

	return "Unknown", "", fmt.Errorf("所有 STUN 服务器检测失败")
}

func detectWithSTUNServer(serverAddr, localIP string) (string, string, error) {
	client := stun.NewClient()
	client.SetServerAddr(serverAddr)

	if localIP != "" {
		client.SetLocalIP(localIP)
	}

	nat, pubIP, err := client.Discover()
	if err != nil {
		return "", "", err
	}

	if nat == stun.NATError || pubIP == nil {
		return "", "", fmt.Errorf("STUN 检测返回无效结果")
	}

	natType, exists := natTypeMap[nat]
	if !exists {
		natType = "Unknown"
	}

	publicIPStr := ""
	if pubIP != nil {
		publicIPStr = strings.Split(pubIP.String(), ":")[0]
	}

	return natType, publicIPStr, nil
}

func ClearNATCache() {
	natCacheLock.Lock()
	defer natCacheLock.Unlock()
	natCache = make(map[string]*natCacheEntry)
}
