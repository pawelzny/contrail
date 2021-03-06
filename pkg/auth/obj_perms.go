package auth

import (
	"time"

	"github.com/databus23/keystone"
)

// identification is struct that describe the identity of resource
type identification struct {
	ID   string `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`
}

// fullIdentification is struct that describe the full identity of resource
type fullIdentification struct {
	identification
	Domain identification `json:"domain"`
}

// token is used in ObjectPerms to store token related information.
type token struct {
	IsDomain  bool               `json:"is_domain"`
	AuthToken string             `json:"auth_token"`
	ExpiresAt string             `json:"expires_at"`
	IssuedAt  string             `json:"issued_at"`
	Version   string             `json:"version"`
	Roles     []identification   `json:"roles"`
	Project   fullIdentification `json:"project"`
	User      fullIdentification `json:"user"`
}

// ObjectPerms holds information get from Keystone module.
type ObjectPerms struct {
	IsGlobalReadOnlyRole bool `json:"is_global_read_only_role"`
	IsCloudAdminRole     bool `json:"is_cloud_admin_role"`
	TokenInfo            struct {
		Token token `json:"token"`
	} `json:"token_info"`
}

// NewObjPerms inits ObjectPerms structure using keystone token
func NewObjPerms(kt *keystone.Token) ObjectPerms {
	if kt == nil {
		return ObjectPerms{}
	}

	// nolint: lll
	// TODO: implement rest of logic: https://github.com/Juniper/contrail-controller/blob/691559e3cbfa9d9db227b4ee55f7eced141c4498/src/config/api-server/vnc_cfg_api_server/vnc_cfg_api_server.py#L2332
	objPerms := ObjectPerms{
		//  part of parameters are set while creating Context in NewContext() method
		TokenInfo: struct {
			Token token `json:"token"`
		}{
			Token: token{
				ExpiresAt: kt.ExpiresAt.Format(time.RFC3339),
				IssuedAt:  kt.IssuedAt.Format(time.RFC3339),
				Version:   "", // TODO(pawel.drapiewski): find the way to get this information if needed
				Roles:     tokenRolesToObjectPermsRoles(kt.Roles),
				Project: fullIdentification{
					identification: identification{
						ID:   kt.Project.ID,
						Name: kt.Project.Name,
					},
					Domain: identification{
						ID:   kt.Project.Domain.ID,
						Name: kt.Project.Domain.Name,
					},
				},
				User: fullIdentification{
					identification: identification{
						ID:   kt.User.ID,
						Name: kt.User.Name,
					},
					Domain: identification{
						ID:   kt.User.Domain.ID,
						Name: kt.User.Domain.Name,
					},
				},
			},
		},
	}

	if kt.Domain != nil {
		objPerms.TokenInfo.Token.IsDomain = kt.Domain.Enabled
	}
	return objPerms
}

func tokenRolesToObjectPermsRoles(tokenRoles []struct {
	ID   string
	Name string
}) []identification {
	var identifications []identification
	for _, role := range tokenRoles {
		identifications = append(identifications, identification{ID: role.ID, Name: role.Name})
	}
	return identifications
}
