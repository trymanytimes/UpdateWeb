package authorization

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/auth/resource"
)

var (
	roleTemplate *resource.RoleTemplate
)

func InitAuthorization() error {
	return readRoleConfig()
}

func readRoleConfig() error {
	roleFile, err := os.Open(config.GetConfig().Server.RoleConf)
	if err != nil {
		return err
	}
	defer roleFile.Close()
	buffer, err := ioutil.ReadAll(roleFile)
	if err != nil {
		return err
	}

	roleTemplate = &resource.RoleTemplate{}
	if err := json.Unmarshal(buffer, roleTemplate); err != nil {
		return err
	}

	return nil
}

func CreateBaseAuthority() map[string]resource.RoleAuthority {
	roleAuthorityMap := make(map[string]resource.RoleAuthority)
	for _, authority := range roleTemplate.Role.Normal.BaseAuthority {
		roleAuthorityMap[authority.Resource] = authority
	}

	CreateViewAuthority([]string{}, roleAuthorityMap)
	CreateDhcpAuthority([]string{}, roleAuthorityMap)
	return roleAuthorityMap
}

func CreateViewAuthority(views []string, roleAuthority map[string]resource.RoleAuthority) {
	for _, authority := range roleTemplate.Role.Normal.DnsAuthority {
		authority.Views = views
		if authority.Filter {
			if len(authority.Views) > 0 {
				authority.Operations = []resource.OperationsType{
					resource.OperationsTypeGET, resource.OperationsTypePUT,
					resource.OperationsTypePOST, resource.OperationsTypeDELETE,
					resource.OperationsTypeACTION}
			} else {
				authority.Operations = []resource.OperationsType{
					resource.OperationsTypeGET}
			}
		}
		roleAuthority[authority.Resource] = authority
	}
}

func CreateDhcpAuthority(plans []string, roleAuthority map[string]resource.RoleAuthority) {
	for _, authority := range roleTemplate.Role.Normal.DhcpAuthority {
		authority.Plans = plans
		if authority.Filter {
			if len(authority.Plans) > 0 {
				authority.Operations = []resource.OperationsType{
					resource.OperationsTypeGET, resource.OperationsTypePUT,
					resource.OperationsTypePOST, resource.OperationsTypeDELETE,
					resource.OperationsTypeACTION}
			} else {
				authority.Operations = []resource.OperationsType{
					resource.OperationsTypeGET}
			}
		}
		roleAuthority[authority.Resource] = authority
	}
}
