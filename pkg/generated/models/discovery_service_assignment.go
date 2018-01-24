package models

// DiscoveryServiceAssignment

import "encoding/json"

// DiscoveryServiceAssignment
type DiscoveryServiceAssignment struct {
	IDPerms     *IdPermsType   `json:"id_perms,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
	Annotations *KeyValuePairs `json:"annotations,omitempty"`
	Perms2      *PermType2     `json:"perms2,omitempty"`
	UUID        string         `json:"uuid,omitempty"`
	ParentUUID  string         `json:"parent_uuid,omitempty"`
	ParentType  string         `json:"parent_type,omitempty"`
	FQName      []string       `json:"fq_name,omitempty"`

	DsaRules []*DsaRule `json:"dsa_rules,omitempty"`
}

// String returns json representation of the object
func (model *DiscoveryServiceAssignment) String() string {
	b, _ := json.Marshal(model)
	return string(b)
}

// MakeDiscoveryServiceAssignment makes DiscoveryServiceAssignment
func MakeDiscoveryServiceAssignment() *DiscoveryServiceAssignment {
	return &DiscoveryServiceAssignment{
		//TODO(nati): Apply default
		UUID:        "",
		ParentUUID:  "",
		ParentType:  "",
		FQName:      []string{},
		IDPerms:     MakeIdPermsType(),
		DisplayName: "",
		Annotations: MakeKeyValuePairs(),
		Perms2:      MakePermType2(),
	}
}

// MakeDiscoveryServiceAssignmentSlice() makes a slice of DiscoveryServiceAssignment
func MakeDiscoveryServiceAssignmentSlice() []*DiscoveryServiceAssignment {
	return []*DiscoveryServiceAssignment{}
}
