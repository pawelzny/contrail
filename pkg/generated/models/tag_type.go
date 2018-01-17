package models

// TagType

import "encoding/json"

// TagType
type TagType struct {
	TagTypeID   U16BitHexInt   `json:"tag_type_id,omitempty"`
	UUID        string         `json:"uuid,omitempty"`
	ParentUUID  string         `json:"parent_uuid,omitempty"`
	FQName      []string       `json:"fq_name,omitempty"`
	Annotations *KeyValuePairs `json:"annotations,omitempty"`
	Perms2      *PermType2     `json:"perms2,omitempty"`
	ParentType  string         `json:"parent_type,omitempty"`
	IDPerms     *IdPermsType   `json:"id_perms,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
}

// String returns json representation of the object
func (model *TagType) String() string {
	b, _ := json.Marshal(model)
	return string(b)
}

// MakeTagType makes TagType
func MakeTagType() *TagType {
	return &TagType{
		//TODO(nati): Apply default
		ParentUUID:  "",
		FQName:      []string{},
		Annotations: MakeKeyValuePairs(),
		Perms2:      MakePermType2(),
		TagTypeID:   MakeU16BitHexInt(),
		UUID:        "",
		DisplayName: "",
		ParentType:  "",
		IDPerms:     MakeIdPermsType(),
	}
}

// MakeTagTypeSlice() makes a slice of TagType
func MakeTagTypeSlice() []*TagType {
	return []*TagType{}
}
