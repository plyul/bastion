package api

type AccessType string

const (
	AccessTypeMandate = "mandate"
	AccessTypeCustom  = "custom"
)

type CreateSessionDTO struct {
	OriginIP              string     `json:"origin_ip"`
	UserName              string     `json:"user_name"`
	TargetProtocolID      int        `json:"target_protocol_id"`
	TargetHost            string     `json:"target_host"`
	TargetPort            int        `json:"target_port"`
	AccessType            AccessType `json:"access_type,omitempty"`
	MandateID             int        `json:"mandate_id,omitempty"`
	CustomTargetNetworkID int        `json:"custom-target-network-id,omitempty"`
	CustomTargetLogin     string     `json:"custom-target-login,omitempty"`
	CustomTargetPassword  string     `json:"custom-target-password,omitempty"`
	CustomTargetPrivKey   string     `json:"custom-target-priv-key,omitempty"`
}

type SessionLocatorDTO struct {
	Token        string `json:"token"`
	NetworkName  string `json:"network_name"`
	Endpoint     string `json:"endpoint"`
	Servicepoint string `json:"servicepoint"`
}

type ReadSessionDTO struct {
	OriginIP       string `json:"origin_ip"`
	TargetNetwork  string `json:"target_network"`
	TargetProtocol string `json:"target_protocol"`
	TargetHost     string `json:"target_host"`
	TargetPort     string `json:"target_port"`
	TargetLogin    string `json:"target-login"`
	TargetPassword string `json:"target-password"`
	TargetPrivKey  string `json:"target-priv-key"`
}

type ReadUserDTO struct {
	User             User              `json:"user"`
	Mandates         []Mandate         `json:"mandates"`
	SessionTemplates []SessionTemplate `json:"session_templates"`
}

type Protocol struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DefaultPort int    `json:"default_port"`
}

type Mandate struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Network struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Endpoint     string `json:"endpoint"`
	Servicepoint string `json:"servicepoint"`
}

type SessionTemplate struct {
	ID                    int        `json:"id"`
	Name                  string     `json:"name"`
	TargetProtocolID      int        `json:"target_protocol_id"`
	TargetHost            string     `json:"target_host"`
	TargetPort            int        `json:"target_port"`
	AccessType            AccessType `json:"access_type"`
	MandateID             int        `json:"mandate_id,omitempty"`
	CustomTargetNetworkID int        `json:"custom_target_network_id,omitempty"`
	CustomTargetLogin     string     `json:"custom_target_login,omitempty"`
	CustomTargetPassword  string     `json:"custom_target_password,omitempty"`
	CustomTargetPrivKey   string     `json:"custom_target_priv_key,omitempty"`
}
