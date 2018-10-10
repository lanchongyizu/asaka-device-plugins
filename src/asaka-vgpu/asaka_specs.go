package main

type AsakaServer struct {
	ServiceIp    string          `json:"service_ip"`
	ServiceName  string          `json:"service_name"`
	ServicePort  int             `json:"service_port"`
	Services     []*AsakaService `json:"services"`
	AllocationId string          `json:"allocation_id"`
}

type AsakaService struct {
	ServedDeviceId string `json:"device_id"`
	DeviceQuota    string `json:"device_quota"`
	Device         Device `json:"device"`
	ServedProtocol string `json:"served_protocol"`
	Occupied       bool   `json:"service_occupied"`
}

type Device struct {
	DeviceId       string       `json:"device_id"`
	Index          string       `json:"device_index"`
	Name           string       `json:"device_name"`
	PlatformVendor string       `json:"device_platform_vendor"`
	PlatformName   string       `json:"device_platform_name"`
	Vendor         string       `json:"device_vendor"`
	Type           string       `json:"device_type"`
	BeloneTo       string       `json:"beloned_user_id"`
	Ip             string       `json:"device_ip"`
	ServicePort    int          `json:"device_port"`
	Protocol       string       `json:"served_protocol"`
	ExtraAttrs     []*ExtraAttr `json:"extra_attributes"`
}

type ExtraAttr struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AsakaError struct {
	ErrorMsg string `json:"Error"`
}
