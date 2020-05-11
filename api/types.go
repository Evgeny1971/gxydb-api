package api

import "time"

type V2Gateway struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Type          string `json:"type"`
	Token         string `json:"token"`
	AdminURL      string `json:"admin_url"`
	AdminPassword string `json:"admin_password"`
}

type V2Config struct {
	Gateways   map[string]map[string]*V2Gateway `json:"gateways"`
	IceServers map[string][]string              `json:"ice_servers"`
}

type V1User struct {
	ID        string `json:"id"`
	Display   string `json:"display"`
	Email     string `json:"email"`
	Group     string `json:"group"`
	IP        string `json:"ip"`
	Janus     string `json:"janus"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	System    string `json:"system"`
	Username  string `json:"username"`
	Room      int    `json:"room"`
	Timestamp int64  `json:"timestamp"`
	Session   int64  `json:"session"`
	Handle    int64  `json:"handle"`
	RFID      int64  `json:"rfid"`
	Camera    bool   `json:"camera"`
	Question  bool   `json:"question"`
	SelfTest  bool   `json:"self_test"`
	SoundTest bool   `json:"sound_test"`
}

type V1RoomInfo struct {
	Room        int    `json:"room"`
	Janus       string `json:"janus"`
	Description string `json:"description"`
}

type V1Room struct {
	V1RoomInfo
	Questions          bool      `json:"questions"`
	NumUsers           int       `json:"num_users"`
	Users              []*V1User `json:"users"`
	firstSessionInRoom time.Time
}

type V1Composite struct {
	VQuad []*V1CompositeRoom `json:"vquad"`
}

type V1CompositeRoom struct {
	V1Room
	Position int `json:"queue"`
}

type V1ProtocolMessageText struct {
	Type   string
	Status bool
	Room   int
	User   V1User
}
