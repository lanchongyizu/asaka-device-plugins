package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

func handleHttpGet(queryUrl string) (string, error) {
	response, err := http.Get(queryUrl)
	if err != nil {
		return "", err
	}

	if response.StatusCode > 299 {
		errorMessage := fmt.Sprintf("Unexpected response codes: %d", response.StatusCode)
		return "", errors.New(errorMessage)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func handleHttpPut(url string, data string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, strings.NewReader(data))
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

type releaseInfo struct {
	allocationId  string
	allocationStr string
}

type AsakaControllerClient struct {
	xaasControllerUri string
	releaseMap        map[int]releaseInfo
}

func NewAsakaControllerClient(controllerUri string) *AsakaControllerClient {
	return &AsakaControllerClient{
		xaasControllerUri: controllerUri,
		releaseMap:        make(map[int]releaseInfo),
	}
}

func (ac *AsakaControllerClient) AllocateVGPU(devs []string) (map[string]string, error) {
	vgpuNeeded := len(devs)
	log.Infof("Request %d VGPUs.", vgpuNeeded)
	if vgpuNeeded > 0 {
		queryUrl := fmt.Sprintf("http://%s/service/asaka_server?served_protocol=CUDA&vgpu_request=%d", ac.xaasControllerUri, vgpuNeeded)
		log.Infof("Query the XaaS Controller for asaka service: %s", queryUrl)
		returnStr, err := handleHttpGet(queryUrl)
		if err != nil {
			return nil, err
		}
		asakaServers, isDone, errorOfHandleResponse := ac.handleHttpResponse(returnStr)

		if errorOfHandleResponse != nil {
			log.Info("Error of handle response: ", errorOfHandleResponse)
			return nil, errorOfHandleResponse
		} else if !isDone {
			log.Info("Error of handle response: ", errorOfHandleResponse)
			return nil, fmt.Errorf("Cannot finsih request GPU resource from XaaS Controller")
		} else if len(asakaServers) == 0 {
			log.Infof("Cannot find enough vGPUs meet the requirment: %d.", vgpuNeeded)
			return nil, fmt.Errorf("Cannot finsih request GPU resource from XaaS Controller")
		}

		envMap := map[string]string{}
		envMap["ASAKA_K8S"] = "1"
		envMap["XaaS-Controller"] = ac.xaasControllerUri
		envMap["CONTROLLER_IP"] = ac.xaasControllerUri
		envMap["ASAKA_CONTROLLER_IP"] = ac.xaasControllerUri

		allocationId := asakaServers[0].AllocationId
		if allocationId != "" {
			log.Infof("Get the allocationId: %s", allocationId)
			allocations := ac.queryVGPUAllocations(allocationId)
			envMap["DEV"] = allocations + ";ALLOCATION_ID=" + allocationId
			ac.confirmedVGPUAllocations(allocationId)
			key := StringsToHash(devs)
			if _, ok := ac.releaseMap[key]; !ok {
				var releaseData releaseInfo
				releaseData.allocationId = allocationId
				releaseData.allocationStr = allocations
				ac.releaseMap[key] = releaseData
			}
		}
		return envMap, nil
	}
	return nil, nil
}

func (ac *AsakaControllerClient) ReleaseVGPU(devs []string) error {
	key := StringsToHash(devs)
	if releaseData, ok := ac.releaseMap[key]; ok {
		log.Infof("Release %s, %s", releaseData.allocationId, releaseData.allocationStr)
		url := fmt.Sprintf("http://%s/device/%s/release", ac.xaasControllerUri, releaseData.allocationId)
		if _, err := handleHttpPut(url, releaseData.allocationStr); err != nil {
			return err
		}
		delete(ac.releaseMap, key)
	}

	return nil
}

func (ac *AsakaControllerClient) handleHttpResponse(returnStr string) ([]AsakaServer, bool, error) {
	if returnStr == "null" {
		return nil, false, errors.New("Not enough asaka vGPU left. Please wait")
	}

	var asakaServers []AsakaServer
	err := json.Unmarshal([]byte(returnStr), &asakaServers)

	if err != nil {
		var asakaError AsakaError
		err := json.Unmarshal([]byte(returnStr), &asakaError)
		return nil, true, err
	}

	return asakaServers, true, nil
}

func (ac *AsakaControllerClient) queryVGPUAllocations(allocationId string) string {
	queryStr := fmt.Sprintf("http://%s/device/%s", ac.xaasControllerUri, allocationId)
	returnStr, err := handleHttpGet(queryStr)
	if err != nil {
		log.Info("Query allocation error: ", err)
	}
	return returnStr
}

func (ac *AsakaControllerClient) confirmedVGPUAllocations(allocationId string) string {
	url := fmt.Sprintf("http://%s/device/%s/allocate", ac.xaasControllerUri, allocationId)
	returnStr, err := handleHttpPut(url, "")

	if err != nil {
		log.Infof("Confirm allocation %s error: %s", allocationId, err)
	}

	return returnStr
}

func (ac *AsakaControllerClient) GetDevices() []*pluginapi.Device {
	queryUrl := fmt.Sprintf("http://%s/device", ac.xaasControllerUri)
	returnStr, err := handleHttpGet(queryUrl)
	if err != nil {
		log.Error(err)
		return nil
	}

	var devices []Device
	if err = json.Unmarshal([]byte(returnStr), &devices); err != nil {
		log.Error(err)
		return nil
	}

	var devs []*pluginapi.Device
	for _, d := range devices {
		vgpuNum := 0
		for _, extra := range d.ExtraAttrs {
			if extra.Key == "vgpu_num" {
				if vgpuNum, err = strconv.Atoi(extra.Value); err != nil {
					log.Error(err)
				}
				break
			}
		}
		for i := 0; i < vgpuNum; i++ {
			vgpuID := d.DeviceId + ":" + strconv.Itoa(i)
			log.Debug("vgpuID: ", vgpuID)
			devs = append(devs, &pluginapi.Device{
				ID:     vgpuID,
				Health: pluginapi.Healthy,
			})
		}
	}

	return devs
}

func (ac *AsakaControllerClient) TestConnection() error {
	queryStr := fmt.Sprintf("http://%s/test", ac.xaasControllerUri)
	_, err := handleHttpGet(queryStr)
	return err
}
