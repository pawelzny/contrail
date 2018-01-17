package models

// NetworkIpam

import "encoding/json"

// NetworkIpam
type NetworkIpam struct {
	ParentType       string           `json:"parent_type,omitempty"`
	IDPerms          *IdPermsType     `json:"id_perms,omitempty"`
	DisplayName      string           `json:"display_name,omitempty"`
	Perms2           *PermType2       `json:"perms2,omitempty"`
	NetworkIpamMGMT  *IpamType        `json:"network_ipam_mgmt,omitempty"`
	IpamSubnets      *IpamSubnets     `json:"ipam_subnets,omitempty"`
	IpamSubnetMethod SubnetMethodType `json:"ipam_subnet_method,omitempty"`
	ParentUUID       string           `json:"parent_uuid,omitempty"`
	UUID             string           `json:"uuid,omitempty"`
	FQName           []string         `json:"fq_name,omitempty"`
	Annotations      *KeyValuePairs   `json:"annotations,omitempty"`

	VirtualDNSRefs []*NetworkIpamVirtualDNSRef `json:"virtual_DNS_refs,omitempty"`
}

// NetworkIpamVirtualDNSRef references each other
type NetworkIpamVirtualDNSRef struct {
	UUID string   `json:"uuid"`
	To   []string `json:"to"` //FQDN

}

// String returns json representation of the object
func (model *NetworkIpam) String() string {
	b, _ := json.Marshal(model)
	return string(b)
}

// MakeNetworkIpam makes NetworkIpam
func MakeNetworkIpam() *NetworkIpam {
	return &NetworkIpam{
		//TODO(nati): Apply default
		UUID:             "",
		FQName:           []string{},
		Annotations:      MakeKeyValuePairs(),
		ParentUUID:       "",
		ParentType:       "",
		IDPerms:          MakeIdPermsType(),
		DisplayName:      "",
		Perms2:           MakePermType2(),
		NetworkIpamMGMT:  MakeIpamType(),
		IpamSubnets:      MakeIpamSubnets(),
		IpamSubnetMethod: MakeSubnetMethodType(),
	}
}

// MakeNetworkIpamSlice() makes a slice of NetworkIpam
func MakeNetworkIpamSlice() []*NetworkIpam {
	return []*NetworkIpam{}
}
