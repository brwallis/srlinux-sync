package main

import (
	"time"

	log "k8s.io/klog"

	"github.com/brwallis/srlinux-sync/internal/agent"
)

const (
	// ndkAddress = "localhost:50053"
	ndkAddress = "unix:///opt/srlinux/var/run/sr_sdk_service_manager:50053"
	agentName  = "dssync"
	yangRoot   = ".dssync"
)

// Global vars
var (
	DSSync agent.Agent
)

func main() {
	log.Infof("Initializing NDK...")
	DSSync = agent.Agent{}
	DSSync.Init(agentName, ndkAddress, yangRoot)

	log.Infof("Starting to receive notifications from NDK...")
	DSSync.Wg.Add(1)
	go DSSync.ReceiveNotifications()

	time.Sleep(2 * time.Second)

	DSSync.Wg.Wait()

	DSSync.GrpcConn.Close()
}
