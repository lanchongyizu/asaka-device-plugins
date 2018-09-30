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
		log.Infof(err.Error())
		return "", err
	}

	defer response.Body.Close()
	if response.StatusCode > 299 {
		errorMessage := fmt.Sprintf("Unexpected response codes: %d", response.StatusCode)
		return "", errors.New(errorMessage)
	}

	body, _ := ioutil.ReadAll(response.Body)
	return_str := string(body)

	return return_str, nil
}

func handleHttpPut(url string, data string) (string, error) {
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, strings.NewReader(data))
	response, err := client.Do(request)
	if err != nil {
		log.Infof(err.Error())
		return "", err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Infof(err.Error())
			return "", err
		}

		return string(contents[:]), nil
	}
}

func getDevices() []*pluginapi.Device {
	var devs []*pluginapi.Device

	url := fmt.Sprintf("http://%s/device", xaasControllerUri)
	response, err := http.Get(url)
	if err != nil {
		log.Infof(err.Error())
	} else if response.StatusCode > 299 {
		log.Infof("Unexpected response code: %d", response.StatusCode)
	} else {
		body, _ := ioutil.ReadAll(response.Body)
		defer response.Body.Close()

		var devices []Device
		err = json.Unmarshal(body, &devices)
		if err == nil {
			for _, d := range devices {
				vgpuNum := 0
				for _, extra := range d.ExtraAttrs {
					if extra.Key == "vgpu_num" {
						if vgpuNum, err = strconv.Atoi(extra.Value); err != nil {
							log.Errorln(err)
						}
						break
					}
				}
				for i := 0; i < vgpuNum; i++ {
					vgpuID := d.DeviceId + ":" + strconv.Itoa(i)
					log.Info("vgpuID: ", vgpuID)
					devs = append(devs, &pluginapi.Device{
						ID:     vgpuID,
						Health: pluginapi.Healthy,
					})
				}
			}
		}
	}

	return devs
}
