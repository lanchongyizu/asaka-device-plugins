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

func getDevices() []*pluginapi.Device {
	queryUrl := fmt.Sprintf("http://%s/device", xaasControllerUri)
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
