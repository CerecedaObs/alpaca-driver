package alpaca

type DeviceInfo struct {
	Name        string `json:"DeviceName"`
	Description string `json:"-"`
	Type        string `json:"DeviceType"`
	Number      int    `json:"DeviceNumber"`
	UniqueID    string `json:"UniqueID"`
}

type DriverInfo struct {
	Name             string
	Version          string
	InterfaceVersion int
}

type StateProperty struct {
	Name  string
	Value interface{}
}

type Device interface {
	DeviceInfo() DeviceInfo
	DriverInfo() DriverInfo
	GetState() []StateProperty

	Connected() bool
	Connecting() bool
	Connect() error
	Disconnect() error
}
