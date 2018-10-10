package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	resourceName    = "asaka/vgpu"
	serverSock      = pluginapi.DevicePluginPath + "asaka-vgpu.sock"
	cudaRequestType = "cudaGPU"
)

// AsakaVgpuDevicePlugin implements the Kubernetes device plugin API
type AsakaVgpuDevicePlugin struct {
	devs   []*pluginapi.Device
	socket string

	stop chan interface{}

	server *grpc.Server
}

// NewAsakaVgpuDevicePlugin returns an initialized AsakaVgpuDevicePlugin
func NewAsakaVgpuDevicePlugin() *AsakaVgpuDevicePlugin {
	return &AsakaVgpuDevicePlugin{
		devs:   getDevices(),
		socket: serverSock,

		stop: make(chan interface{}),
	}
}

func (m *AsakaVgpuDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// Start starts the gRPC server of the device plugin
func (m *AsakaVgpuDevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(m.server, m)

	go m.server.Serve(sock)

	// Wait for server to start by launching a blocking connexion
	conn, err := dial(m.socket, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

// Stop stops the gRPC server
func (m *AsakaVgpuDevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}

	m.server.Stop()
	m.server = nil
	close(m.stop)

	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *AsakaVgpuDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(m.socket),
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

// ListAndWatch lists devices and update that list per second
func (m *AsakaVgpuDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})

	for {
		select {
		case <-m.stop:
			return nil
		case <-time.After(time.Second):
			m.devs = getDevices()
			s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})
		}
	}
}

// Allocate which return list of devices.
func (m *AsakaVgpuDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		log.Info("Request DevicesIDs: ", strings.Join(req.DevicesIDs, ","))

		envMap := map[string]string{}
		envMap["ASAKA_K8S"] = "1"
		envMap["XaaS-Controller"] = xaasControllerUri
		envMap["CONTROLLER_IP"] = xaasControllerUri
		envMap["ASAKA_CONTROLLER_IP"] = xaasControllerUri

		vgpuNeeded := len(req.DevicesIDs)
		if vgpuNeeded > 0 {
			queryUrl := fmt.Sprintf("http://%s/service/asaka_server?served_protocol=CUDA&vgpu_request=%d", xaasControllerUri, vgpuNeeded)
			log.Infof("Query the XaaS Controller for asaka service: %s", queryUrl)
			asakaServers, isDone, errorOfHandleResponse := m.handleHttpResponse(cudaRequestType, queryUrl)

			if errorOfHandleResponse != nil {
				log.Info("Error of handle response: ", errorOfHandleResponse)
				return nil, errorOfHandleResponse
			} else if !isDone {
				log.Info("Error of handle response: ", errorOfHandleResponse)
				return nil, fmt.Errorf("Cannot finsih request GPU resource from XaaS Controller")
			} else if len(asakaServers) == 0 {
				log.Infof("Cannot find enough vGPUs meet the requirment: %d.", vgpuNeeded)
				return nil, fmt.Errorf("Cannot finsih request GPU resource from XaaS Controller")
			} else {
				allocationId := asakaServers[0].AllocationId
				if allocationId != "" {
					log.Infof("Get the allocationId: %s", allocationId)
					allocations := m.queryVGPUAllocations(allocationId)
					envMap["DEV"] = allocations + ";ALLOCATION_ID=" + allocationId
					m.confirmedVGPUAllocations(allocationId)
					// if _, ok := releaseMap[pod.SelfLink]; !ok {
					// 	var releaseData releaseInfo
					// 	releaseData.allocationId = allocationId
					// 	releaseData.allocationStr = allocations
					// 	releaseMap[pod.SelfLink] = releaseData
					// }
				}

				stringEnv, _ := json.Marshal(envMap)
				log.Info("Set the env for the container: ", string(stringEnv))
			}
		}

		response := pluginapi.ContainerAllocateResponse{
			Envs: envMap,
		}

		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}

	return &responses, nil
}

func (m *AsakaVgpuDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *AsakaVgpuDevicePlugin) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *AsakaVgpuDevicePlugin) Serve() error {
	err := m.Start()
	if err != nil {
		log.Infof("Could not start device plugin: %s", err)
		return err
	}
	log.Info("Starting to serve on ", m.socket)

	err = m.Register(pluginapi.KubeletSocket, resourceName)
	if err != nil {
		log.Infof("Could not register device plugin: %s", err)
		m.Stop()
		return err
	}
	log.Info("Registered device plugin with Kubelet")

	return nil
}

func (m *AsakaVgpuDevicePlugin) handleHttpResponse(queryType string, queryUrl string) ([]AsakaServer, bool, error) {
	returnStr, err := handleHttpGet(queryUrl)
	if err != nil {
		log.Info(err)
		return nil, false, err
	}

	if returnStr == "null" {
		return nil, false, errors.New("Not enough asaka vGPU left. Please wait")
	}

	var asakaServers []AsakaServer
	err = json.Unmarshal([]byte(returnStr), &asakaServers)

	if err != nil {
		var asakaError AsakaError
		err := json.Unmarshal([]byte(returnStr), &asakaError)
		return nil, true, err
	}

	return asakaServers, true, nil
}

func (m *AsakaVgpuDevicePlugin) queryVGPUAllocations(allocationId string) string {
	queryStr := fmt.Sprintf("http://%s/device/%s", xaasControllerUri, allocationId)
	returnStr, err := handleHttpGet(queryStr)
	if err != nil {
		log.Info("Query allocation error: ", err)
	}
	return returnStr
}

func (m *AsakaVgpuDevicePlugin) confirmedVGPUAllocations(allocationId string) string {
	url := fmt.Sprintf("http://%s/device/%s/allocate", xaasControllerUri, allocationId)
	returnStr, err := handleHttpPut(url, "")

	if err != nil {
		log.Infof("Confirm allocation %s error: %s", allocationId, err)
	}

	return returnStr
}
