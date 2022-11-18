package main

type Switch struct {
	Hostname     string
	Ifaces       []Ethernet
	ControlVlans []Vlan
	Gateway      string
}

type Vlan struct {
	VlanId string
	IP     string
}

type Ethernet struct {
	EthName    string
	Vlan       string
	PortRole   string
	TrunkVlans []string
}
