package main

import (
	"os"
	"syscall"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

var asakaControllerClient *AsakaControllerClient

func initLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

func initControllerClient() {
	xaasControllerUri := os.Getenv("XAAS_CONTROLLER_URI")
	if xaasControllerUri == "" {
		log.Fatal("XAAS_CONTROLLER_URI can't be empty.")
	}

	log.Infof("XaaS Controller URI: %s", xaasControllerUri)
	asakaControllerClient = NewAsakaControllerClient(xaasControllerUri)
	if err := asakaControllerClient.TestConnection(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	initLogger()
	initControllerClient()
}

func main() {
	log.Info("Starting FS watcher.")
	watcher, err := newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		log.Info("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()

	log.Info("Starting OS watcher.")
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
				log.Info("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
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
				log.Info("Received SIGHUP, restarting.")
				restart = true
			default:
				log.Infof("Received signal \"%v\", shutting down.", s)
				devicePlugin.Stop()
				break L
			}
		}
	}
}
