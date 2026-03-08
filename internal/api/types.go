package api

// Pod represents a Runpod pod.
type Pod struct {
	ID               string   `json:"id" yaml:"id"`
	Name             string   `json:"name" yaml:"name"`
	ImageName        string   `json:"imageName" yaml:"imageName"`
	DesiredStatus    string   `json:"desiredStatus" yaml:"desiredStatus"`
	PodType          string   `json:"podType" yaml:"podType"`
	GPUCount         int      `json:"gpuCount" yaml:"gpuCount"`
	VolumeInGB       float64  `json:"volumeInGb" yaml:"volumeInGb"`
	ContainerDiskGB  float64  `json:"containerDiskInGb" yaml:"containerDiskInGb"`
	MemoryInGB       float64  `json:"memoryInGb" yaml:"memoryInGb"`
	VCPUCount        float64  `json:"vcpuCount" yaml:"vcpuCount"`
	CostPerHr        float64  `json:"costPerHr" yaml:"costPerHr"`
	VolumeMountPath  string   `json:"volumeMountPath" yaml:"volumeMountPath"`
	Ports            string   `json:"ports" yaml:"ports"`
	DockerArgs       string   `json:"dockerArgs" yaml:"dockerArgs"`
	Env              []EnvVar `json:"env" yaml:"env"`
	TemplateID       string   `json:"templateId" yaml:"templateId"`
	MachineID        string   `json:"machineId" yaml:"machineId"`
	UptimeSeconds    int      `json:"uptimeSeconds" yaml:"uptimeSeconds"`
	Locked           bool     `json:"locked" yaml:"locked"`
	CreatedAt        string   `json:"createdAt" yaml:"createdAt"`
	LastStartedAt    string   `json:"lastStartedAt" yaml:"lastStartedAt"`
	LastStatusChange string   `json:"lastStatusChange" yaml:"lastStatusChange"`
}

// EnvVar represents an environment variable.
type EnvVar struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

// Endpoint represents a serverless endpoint.
type Endpoint struct {
	ID            string `json:"id" yaml:"id"`
	Name          string `json:"name" yaml:"name"`
	TemplateID    string `json:"templateId" yaml:"templateId"`
	GPUIDs        string `json:"gpuIds" yaml:"gpuIds"`
	WorkersMin    int    `json:"workersMin" yaml:"workersMin"`
	WorkersMax    int    `json:"workersMax" yaml:"workersMax"`
	IdleTimeout   int    `json:"idleTimeout" yaml:"idleTimeout"`
	NetworkVolume string `json:"networkVolumeId" yaml:"networkVolumeId"`
}

// Template represents a pod template.
type Template struct {
	ID             string   `json:"id" yaml:"id"`
	Name           string   `json:"name" yaml:"name"`
	ImageName      string   `json:"imageName" yaml:"imageName"`
	DockerStartCmd string   `json:"dockerStartCmd" yaml:"dockerStartCmd"`
	ContainerDisk  float64  `json:"containerDiskInGb" yaml:"containerDiskInGb"`
	VolumeMountPath string  `json:"volumeMountPath" yaml:"volumeMountPath"`
	Ports          string   `json:"ports" yaml:"ports"`
	Env            []EnvVar `json:"env" yaml:"env"`
	IsPublic       bool     `json:"isPublic" yaml:"isPublic"`
	IsServerless   bool     `json:"isServerless" yaml:"isServerless"`
}

// Volume represents a network volume.
type Volume struct {
	ID           string  `json:"id" yaml:"id"`
	Name         string  `json:"name" yaml:"name"`
	Size         float64 `json:"size" yaml:"size"`
	DataCenterID string  `json:"dataCenterId" yaml:"dataCenterId"`
}

// Registry represents a container registry credential.
type Registry struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	URL      string `json:"url" yaml:"url"`
	Username string `json:"username" yaml:"username"`
}

// GPUType represents a GPU type.
type GPUType struct {
	ID                string  `json:"id" yaml:"id"`
	DisplayName       string  `json:"displayName" yaml:"displayName"`
	MemoryInGB        int     `json:"memoryInGb" yaml:"memoryInGb"`
	SecureCloud       bool    `json:"secureCloud" yaml:"secureCloud"`
	CommunityCloud    bool    `json:"communityCloud" yaml:"communityCloud"`
	LowestPrice       *Price  `json:"lowestPrice" yaml:"lowestPrice"`
}

// Price represents pricing info.
type Price struct {
	MinimumBidPrice   float64 `json:"minimumBidPrice" yaml:"minimumBidPrice"`
	Uninterruptable   float64 `json:"uninterruptablePrice" yaml:"uninterruptablePrice"`
}

// Secret represents an API secret.
type Secret struct {
	Name      string `json:"name" yaml:"name"`
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
}

// PortMapping represents a pod's port mapping at runtime.
type PortMapping struct {
	IP          string `json:"ip" yaml:"ip"`
	IsIPPublic  bool   `json:"isIpPublic" yaml:"isIpPublic"`
	PrivatePort int    `json:"privatePort" yaml:"privatePort"`
	PublicPort  int    `json:"publicPort" yaml:"publicPort"`
	Type        string `json:"type" yaml:"type"`
}

// BillingInfo represents billing data.
type BillingInfo struct {
	ID        string  `json:"id" yaml:"id"`
	Name      string  `json:"name" yaml:"name"`
	CostPerHr float64 `json:"costPerHr" yaml:"costPerHr"`
	TotalCost float64 `json:"totalCost" yaml:"totalCost"`
}
