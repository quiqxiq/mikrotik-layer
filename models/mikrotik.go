package models

type Interface struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Running    bool   `json:"running"`
	Disabled   bool   `json:"disabled"`
	RxBytes    string `json:"rx-bytes,omitempty"`
	TxBytes    string `json:"tx-bytes,omitempty"`
	RxPackets  string `json:"rx-packets,omitempty"`
	TxPackets  string `json:"tx-packets,omitempty"`
}

type Address struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Interface string `json:"interface"`
	Network   string `json:"network"`
	Disabled  bool   `json:"disabled"`
}

type Queue struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Target   string `json:"target"`
	MaxLimit string `json:"max-limit"`
	BurstLimit string `json:"burst-limit"`
	Disabled bool   `json:"disabled"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}