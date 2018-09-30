package main

import (
	"os"
	"syscall"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type releaseInfo struct {
	allocationId  string
	allocationStr string
}

var (
	xaasControllerUri = os.Getenv("XAAS_CONTROLLER_URI")
)

var releaseMap map[string]releaseInfo

func main() {
	log.Infof("XaaS Controller URI: %s", xaasControllerUri)
	releaseMap = make(map[string]releaseInfo)

	log.Infoln("Fetching devices.")
	if len(getDevices()) == 0 {
		log.Infoln("No devices found. Waiting indefinitely.")
		select {}
	}

	log.Infoln("Starting FS watcher.")
	watcher, err := newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		log.Infoln("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()

	log.Infoln("Starting OS watcher.")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	restart := true
	var devicePlugin *AsakaVgpuDevicePlugin

L:
	for {
		if restart {
			if devicePlugin != nil {
				devicePlugin.Stop()
			}

			devicePlugin = NewAsakaVgpuDevicePlugin()
			if err := devicePlugin.Serve(); err != nil {
				log.Infoln("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
			} else {
				restart = false
			}
		}

		select {
		case event := <-watcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				log.Infof("inotify: %s created, restarting.", pluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			log.Infof("inotify: %s", err)

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				log.Infoln("Received SIGHUP, restarting.")
				restart = true
			default:
				log.Infof("Received signal \"%v\", shutting down.", s)
				devicePlugin.Stop()
				break L
			}
		}
	}
}
