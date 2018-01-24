package models

// ApplicationPolicySet

import "encoding/json"

// ApplicationPolicySet
type ApplicationPolicySet struct {
	DisplayName     string         `json:"display_name,omitempty"`
	UUID            string         `json:"uuid,omitempty"`
	FQName          []string       `json:"fq_name,omitempty"`
	IDPerms         *IdPermsType   `json:"id_perms,omitempty"`
	AllApplications bool           `json:"all_applications"`
	Perms2          *PermType2     `json:"perms2,omitempty"`
	ParentUUID      string         `json:"parent_uuid,omitempty"`
	ParentType      string         `json:"parent_type,omitempty"`
	Annotations     *KeyValuePairs `json:"annotations,omitempty"`

	FirewallPolicyRefs      []*ApplicationPolicySetFirewallPolicyRef      `json:"firewall_policy_refs,omitempty"`
	GlobalVrouterConfigRefs []*ApplicationPolicySetGlobalVrouterConfigRef `json:"global_vrouter_config_refs,omitempty"`
}

// ApplicationPolicySetGlobalVrouterConfigRef references each other
type ApplicationPolicySetGlobalVrouterConfigRef struct {
	UUID string   `json:"uuid"`
	To   []string `json:"to"` //FQDN

}

// ApplicationPolicySetFirewallPolicyRef references each other
type ApplicationPolicySetFirewallPolicyRef struct {
	UUID string   `json:"uuid"`
	To   []string `json:"to"` //FQDN

	Attr *FirewallSequence
}

// String returns json representation of the object
func (model *ApplicationPolicySet) String() string {
	b, _ := json.Marshal(model)
	return string(b)
}

// MakeApplicationPolicySet makes ApplicationPolicySet
func MakeApplicationPolicySet() *ApplicationPolicySet {
	return &ApplicationPolicySet{
		//TODO(nati): Apply default
		Annotations:     MakeKeyValuePairs(),
		Perms2:          MakePermType2(),
		ParentUUID:      "",
		ParentType:      "",
		AllApplications: false,
		DisplayName:     "",
		UUID:            "",
		FQName:          []string{},
		IDPerms:         MakeIdPermsType(),
	}
}

// MakeApplicationPolicySetSlice() makes a slice of ApplicationPolicySet
func MakeApplicationPolicySetSlice() []*ApplicationPolicySet {
	return []*ApplicationPolicySet{}
}
