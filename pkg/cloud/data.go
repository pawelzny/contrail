package cloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/Juniper/contrail/pkg/apisrv/client"
	"github.com/Juniper/contrail/pkg/models"
	"github.com/Juniper/contrail/pkg/services"
)

const (
	gatewayRole = "gateway"
	computeRole = "compute"
)

//Data for cloud provider data
type Data struct {
	cloud          *Cloud
	info           *models.Cloud
	credentials    []*models.Credential
	users          []*models.CloudUser
	subnets        []*subnetData
	securityGroups []*sgData
	providers      []*providerData
	instances      []*instanceData
	tors           []*torData
	delRequest     bool
}

type apiServer struct {
	client *client.HTTP
	ctx    context.Context
}

type providerData struct {
	parentCloud *Data
	info        *models.CloudProvider
	regions     []*regionData
	apiServer
}

type regionData struct {
	parentProvider *providerData
	info           *models.CloudRegion
	virtualClouds  []*virtualCloudData
	apiServer
}

type virtualCloudData struct {
	parentRegion *regionData
	info         *models.VirtualCloud
	sgs          []*sgData
	instances    []*instanceData
	tors         []*torData
	subnets      []*subnetData
	apiServer
}

type instanceData struct {
	parentVC      *virtualCloudData
	info          *models.Node
	roles         []string
	protocolsMode []string
	provision     string
	pvtIntf       *models.Port
	gateway       string
	services      []string
	apiServer
}

type torData struct {
	parentVC               *virtualCloudData
	info                   *models.PhysicalRouter
	provision              string
	autonomousSystemNumber int
	interfaceNames         []string
	privateSubnets         []string
	apiServer
}

type subnetData struct {
	parentVC *virtualCloudData
	info     *models.CloudPrivateSubnet
	apiServer
}

type sgData struct {
	parentVC *virtualCloudData
	info     *models.CloudSecurityGroup
	apiServer
}

func (s *subnetData) getPvtSubnetObject() (*models.CloudPrivateSubnet, error) {

	request := new(services.GetCloudPrivateSubnetRequest)
	request.ID = s.info.UUID

	subnetResp, err := s.client.GetCloudPrivateSubnet(s.ctx, request)
	if err != nil {
		return nil, err
	}
	return subnetResp.GetCloudPrivateSubnet(), nil
}

func (v *virtualCloudData) newSubnet(subnet *models.CloudPrivateSubnet) (*subnetData, error) {

	s := &subnetData{
		parentVC: v,
		info:     subnet,
		apiServer: apiServer{
			client: v.client,
			ctx:    v.ctx,
		},
	}

	subnetObj, err := s.getPvtSubnetObject()
	if err != nil {
		return nil, err
	}

	s.info = subnetObj
	return s, nil
}

func (v *virtualCloudData) updateSubnets() error {

	for _, subnet := range v.info.CloudPrivateSubnets {
		newSubnet, err := v.newSubnet(subnet)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}
		v.subnets = append(v.subnets, newSubnet)
	}

	data := v.parentRegion.parentProvider.parentCloud
	data.subnets = append(data.subnets, v.subnets...)
	return nil
}

func (i *instanceData) getNodeObject() (*models.Node, error) {

	request := new(services.GetNodeRequest)
	request.ID = i.info.UUID

	instResp, err := i.client.GetNode(i.ctx, request)
	if err != nil {
		return nil, err
	}
	return instResp.GetNode(), nil
}

func (i *torData) getTorObject() (*models.PhysicalRouter, error) {

	request := new(services.GetPhysicalRouterRequest)
	request.ID = i.info.UUID

	torResp, err := i.client.GetPhysicalRouter(i.ctx, request)
	if err != nil {
		return nil, err
	}
	return torResp.GetPhysicalRouter(), nil
}

// nolint: gocyclo
func (v *virtualCloudData) newInstance(instance *models.Node,
	isDelRequest bool) (*instanceData, error) {

	inst := &instanceData{
		parentVC: v,
		info:     instance,
		apiServer: apiServer{
			client: v.client,
			ctx:    v.ctx,
		},
	}

	instObj, err := inst.getNodeObject()
	if err != nil {
		return nil, err
	}
	inst.info = instObj

	data := v.parentRegion.parentProvider.parentCloud
	if data.isCloudPrivate() {

		i := inst

		if i.info.ContrailVrouterNodeBackRefs != nil && i.info.KubernetesNodeBackRefs != nil {
			i.roles = append(i.roles, "compute_node")
		} else if i.info.ContrailVrouterNodeBackRefs != nil {
			i.roles = append(i.roles, "vrouter")
		}
		if i.info.ContrailConfigNodeBackRefs != nil || i.info.ContrailControlNodeBackRefs != nil {
			i.roles = append(i.roles, "controller")
			i.provision = strconv.FormatBool(false)
		}

		if i.info.KubernetesMasterNodeBackRefs != nil {
			i.roles = append(i.roles, "k8s_master")
		}

		if i.info.ContrailMulticloudGWNodeBackRefs != nil {
			i.roles = append(i.roles, "gateway")
		}

		err = inst.updatePvtIntf(isDelRequest)
		if err != nil {
			return nil, err
		}

		if inst.info.OpenstackComputeNodeBackRefs != nil {
			inst.provision = strconv.FormatBool(false)
		}
	}

	if inst.provision == "" {
		inst.provision = strconv.FormatBool(true)
	}

	if inst.info.ContrailMulticloudGWNodeBackRefs != nil {
		err := inst.updateProtoModes(isDelRequest) //nolint: govet
		if err != nil {
			return nil, err
		}
		err = inst.updateMCGWServices()
		if err != nil {
			return nil, err
		}
		if v.parentRegion.parentProvider.parentCloud.isCloudPrivate() {
			err = inst.updateVrouterGW(gatewayRole, isDelRequest)
			if err != nil {
				return nil, err
			}
		}
	}

	if inst.info.ContrailVrouterNodeBackRefs != nil {
		if v.parentRegion.parentProvider.parentCloud.isCloudPrivate() {
			err = inst.updateVrouterGW(computeRole, isDelRequest)
			if err != nil {
				return nil, err
			}
		}
	}

	return inst, nil
}

func (v *virtualCloudData) newTorInstance(p *models.PhysicalRouter) (tor *torData, err error) {
	data := v.parentRegion.parentProvider.parentCloud
	if !data.isCloudPrivate() {
		return nil, nil
	}
	tor = &torData{
		parentVC: v,
		info:     p,
		apiServer: apiServer{
			client: v.client,
			ctx:    v.ctx,
		},
	}
	tor.info, err = tor.getTorObject()
	if err != nil {
		return nil, err
	}
	var k []*models.KeyValuePair
	if a := tor.info.GetAnnotations(); a != nil {
		k = a.GetKeyValuePair()
	}
	for _, keyValuePair := range k {
		switch keyValuePair.Key {
		case "autonomous_system":
			tor.autonomousSystemNumber, err = strconv.Atoi(keyValuePair.Value)
		case "interface":
			tor.interfaceNames = strings.Split(keyValuePair.Value, ",")
		case "private_subnet":
			tor.privateSubnets = strings.Split(keyValuePair.Value, ",")
		}
	}

	if tor.provision == "" {
		tor.provision = strconv.FormatBool(true)
	}
	return tor, nil
}

func (v *virtualCloudData) updateInstances(isdelRequest bool) error {

	nodes, err := v.getInstancesWithTag(v.info.TagRefs, isdelRequest)
	if err != nil {
		return err
	}

	for _, instance := range nodes {
		newI, err := v.newInstance(instance, isdelRequest)
		if err != nil {
			return err
		}
		v.instances = append(v.instances, newI)
	}

	data := v.parentRegion.parentProvider.parentCloud
	data.instances = append(data.instances, v.instances...)
	return nil
}

func (v *virtualCloudData) updateTorInstances() error {

	physicalRouters, err := v.getTorInstancesWithTag(v.info.TagRefs)
	if err != nil {
		return err
	}

	for _, physicalRouter := range physicalRouters {
		newI, err := v.newTorInstance(physicalRouter)
		if err != nil {
			return err
		}
		v.tors = append(v.tors, newI)
	}

	data := v.parentRegion.parentProvider.parentCloud
	data.tors = append(data.tors, v.tors...)
	return nil
}

func (sg *sgData) getSGObject() (*models.CloudSecurityGroup, error) {

	request := new(services.GetCloudSecurityGroupRequest)
	request.ID = sg.info.UUID

	sgResp, err := sg.client.GetCloudSecurityGroup(sg.ctx, request)
	if err != nil {
		return nil, err
	}
	return sgResp.GetCloudSecurityGroup(), nil
}

func (v *virtualCloudData) getInstancesWithTag(tagRefs []*models.VirtualCloudTagRef,
	isDelRequest bool) ([]*models.Node, error) {
	var nodesOfVC []*models.Node

	for _, tag := range tagRefs {
		tagResp, err := v.client.GetTag(v.ctx, &services.GetTagRequest{ID: tag.UUID})
		if err != nil {
			return nil, err
		}
		nodesOfVC = append(nodesOfVC, tagResp.Tag.NodeBackRefs...)
	}
	if len(nodesOfVC) == 0 && !isDelRequest {
		return nil, errors.New("virtual cloud tag is not used by any nodes")
	}

	for i, node := range nodesOfVC {
		nodeResp, err := v.client.GetNode(v.ctx,
			&services.GetNodeRequest{
				ID: node.UUID,
			},
		)
		if err != nil {
			return nil, err
		}
		nodesOfVC[i] = nodeResp.Node
	}
	return nodesOfVC, nil
}

func (v *virtualCloudData) getTorInstancesWithTag(
	tagRefs []*models.VirtualCloudTagRef) ([]*models.PhysicalRouter, error) {
	var torOfVC []*models.PhysicalRouter

	for _, tag := range tagRefs {
		tagResp, err := v.client.GetTag(v.ctx, &services.GetTagRequest{ID: tag.UUID})
		if err != nil {
			return nil, err
		}
		torOfVC = append(torOfVC, tagResp.Tag.PhysicalRouterBackRefs...)
	}
	return torOfVC, nil
}

func (v *virtualCloudData) newSG(mSG *models.CloudSecurityGroup) (*sgData, error) {

	sg := &sgData{
		parentVC: v,
		info:     mSG,
		apiServer: apiServer{
			client: v.client,
			ctx:    v.ctx,
		},
	}

	sgObj, err := sg.getSGObject()
	if err != nil {
		return nil, err
	}

	sg.info = sgObj
	return sg, nil
}

func (v *virtualCloudData) updateSGs() error {

	for _, sg := range v.info.CloudSecurityGroups {
		newSG, err := v.newSG(sg)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}
		v.sgs = append(v.sgs, newSG)
	}

	data := v.parentRegion.parentProvider.parentCloud
	data.securityGroups = append(data.securityGroups, v.sgs...)
	return nil
}

func (v *virtualCloudData) getVCloudObject() (*models.VirtualCloud, error) {

	request := new(services.GetVirtualCloudRequest)
	request.ID = v.info.UUID

	vCloudResp, err := v.client.GetVirtualCloud(v.ctx, request)
	if err != nil {
		return nil, err
	}
	return vCloudResp.GetVirtualCloud(), nil
}

func (v *virtualCloudData) updateNodeWithTag(
	nodeUUID string, nTagRefs []*models.NodeTagRef) error {

	getNodeResp, err := v.client.GetNode(v.ctx,
		&services.GetNodeRequest{
			ID: nodeUUID,
		},
	)
	if err != nil {
		return err
	}

	for _, nTagRef := range nTagRefs {
		getNodeResp.Node.AddTagRef(nTagRef)
	}

	_, err = v.client.UpdateNode(v.ctx,
		&services.UpdateNodeRequest{
			Node: getNodeResp.Node,
		},
	)
	return err
}

func (v *virtualCloudData) updateControlNodeWithTag(
	controlNodes []*models.ContrailControlNode) error {

	if controlNodes == nil {
		return fmt.Errorf("cluster does not have control nodes")
	}

	for _, controlNode := range controlNodes {
		getControlResp, err := v.client.GetContrailControlNode(v.ctx,
			&services.GetContrailControlNodeRequest{
				ID: controlNode.UUID,
			},
		)
		if err != nil {
			return err
		}
		for _, nodeRef := range getControlResp.ContrailControlNode.NodeRefs {
			var nodeTagRefs []*models.NodeTagRef
			for _, vTagRef := range v.info.TagRefs {
				nodeTagRef := new(models.NodeTagRef)
				nodeTagRef.UUID = vTagRef.UUID
				nodeTagRef.To = vTagRef.To
				nodeTagRef.Href = vTagRef.Href
				nodeTagRefs = append(nodeTagRefs, nodeTagRef)
			}
			err := v.updateNodeWithTag(nodeRef.UUID, nodeTagRefs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *virtualCloudData) updateConfigNodeWithTag(
	configNodes []*models.ContrailConfigNode) error {

	if configNodes == nil {
		return fmt.Errorf("cluster does not have config nodes")
	}

	for _, configNode := range configNodes {
		getConfigResp, err := v.client.GetContrailConfigNode(v.ctx,
			&services.GetContrailConfigNodeRequest{
				ID: configNode.UUID,
			},
		)
		if err != nil {
			return err
		}
		for _, nodeRef := range getConfigResp.ContrailConfigNode.NodeRefs {
			var nodeTagRefs []*models.NodeTagRef
			for _, vTagRef := range v.info.TagRefs {
				nodeTagRef := new(models.NodeTagRef)
				nodeTagRef.UUID = vTagRef.UUID
				nodeTagRef.To = vTagRef.To
				nodeTagRef.Href = vTagRef.Href
				nodeTagRefs = append(nodeTagRefs, nodeTagRef)
			}
			err := v.updateNodeWithTag(nodeRef.UUID, nodeTagRefs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *virtualCloudData) updateVrouterNodeWithTag(
	vrouterNodes []*models.ContrailVrouterNode) error {

	if vrouterNodes == nil {
		return fmt.Errorf("cluster does not have vrouter nodes")
	}

	for _, vrouterNode := range vrouterNodes {
		getVrouterResp, err := v.client.GetContrailVrouterNode(v.ctx,
			&services.GetContrailVrouterNodeRequest{
				ID: vrouterNode.UUID,
			},
		)
		if err != nil {
			return err
		}
		for _, nodeRef := range getVrouterResp.ContrailVrouterNode.NodeRefs {
			var nodeTagRefs []*models.NodeTagRef
			for _, vTagRef := range v.info.TagRefs {
				nodeTagRef := new(models.NodeTagRef)
				nodeTagRef.UUID = vTagRef.UUID
				nodeTagRef.To = vTagRef.To
				nodeTagRef.Href = vTagRef.Href
				nodeTagRefs = append(nodeTagRefs, nodeTagRef)
			}
			err := v.updateNodeWithTag(nodeRef.UUID, nodeTagRefs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *virtualCloudData) updateClusterNodeWithTag(
	mcGWNode *models.ContrailMulticloudGWNode) error {

	ccResp, err := v.client.GetContrailCluster(v.ctx,
		&services.GetContrailClusterRequest{
			ID: mcGWNode.ParentUUID,
		},
	)

	if err != nil {
		return err
	}

	contrailCluster := ccResp.GetContrailCluster()
	if contrailCluster.ContrailControlNodes != nil {
		err = v.updateControlNodeWithTag(contrailCluster.GetContrailControlNodes())
		if err != nil {
			return err
		}
	}
	if contrailCluster.ContrailConfigNodes != nil {
		err = v.updateConfigNodeWithTag(contrailCluster.GetContrailConfigNodes())
		if err != nil {
			return err
		}
	}
	if contrailCluster.ContrailConfigNodes == nil &&
		contrailCluster.ContrailControlNodes == nil {
		return fmt.Errorf("cluster %s does not have control nodes or config nodes",
			contrailCluster.UUID)
	}

	return v.updateVrouterNodeWithTag(contrailCluster.GetContrailVrouterNodes())

}

func (v *virtualCloudData) getMCGWNodeRole(
	instances []*models.Node) (*models.ContrailMulticloudGWNode, error) {

	for _, i := range instances {
		if i.GetContrailMulticloudGWNodeBackRefs() != nil {
			mcGWNodeRefs := i.GetContrailMulticloudGWNodeBackRefs()
			for _, m := range mcGWNodeRefs {
				getMCGWResp, err := v.client.GetContrailMulticloudGWNode(v.ctx,
					&services.GetContrailMulticloudGWNodeRequest{
						ID: m.UUID,
					},
				)
				return getMCGWResp.GetContrailMulticloudGWNode(), err
			}
		}
	}
	return nil, fmt.Errorf(
		"instances list does not have multicloud gw node back refs")
}

func (v *virtualCloudData) getTagsAndUpdateClusterNodes() error {

	instances, err := v.getInstancesWithTag(v.info.TagRefs,
		v.parentRegion.parentProvider.parentCloud.delRequest)
	if err != nil {
		return err
	}

	mcGWNodeRole, err := v.getMCGWNodeRole(instances)
	if err != nil {
		return err
	}

	return v.updateClusterNodeWithTag(mcGWNodeRole)

}

func (r *regionData) newVCloud(vCloud *models.VirtualCloud) (*virtualCloudData, error) {

	vc := &virtualCloudData{
		parentRegion: r,
		info:         vCloud,
		apiServer: apiServer{
			client: r.client,
			ctx:    r.ctx,
		},
	}

	vCloudObj, err := vc.getVCloudObject()
	vc.info = vCloudObj

	if err != nil {
		return nil, err
	}

	return vc, nil
}

// nolint: gocyclo
func (r *regionData) updateVClouds() error {

	for _, vc := range r.info.VirtualClouds {
		newVC, err := r.newVCloud(vc)
		if err != nil {
			return err
		}

		err = newVC.updateSGs()
		if err != nil {
			return err
		}

		isDelRequest := r.parentProvider.parentCloud.delRequest

		if r.parentProvider.parentCloud.isCloudPrivate() && !isDelRequest {
			err = newVC.getTagsAndUpdateClusterNodes()
			if err != nil {
				return err
			}
		}

		err = newVC.updateInstances(isDelRequest)
		if err != nil {
			return err
		}

		if r.parentProvider.parentCloud.isCloudPrivate() {
			err = newVC.updateTorInstances()
			if err != nil {
				return err
			}
		}

		err = newVC.updateSubnets()
		if err != nil {
			return err
		}

		r.virtualClouds = append(r.virtualClouds, newVC)
	}
	return nil
}

func (r *regionData) getRegionObject() (*models.CloudRegion, error) {

	request := new(services.GetCloudRegionRequest)
	request.ID = r.info.UUID

	regResp, err := r.client.GetCloudRegion(r.ctx, request)
	if err != nil {
		return nil, err
	}

	return regResp.GetCloudRegion(), nil

}

func (p *providerData) newRegion(region *models.CloudRegion) (*regionData, error) {

	reg := &regionData{
		parentProvider: p,
		info:           region,
		apiServer: apiServer{
			client: p.client,
			ctx:    p.ctx,
		},
	}

	regObj, err := reg.getRegionObject()
	reg.info = regObj

	if err != nil {
		return nil, err
	}

	return reg, nil
}

func (p *providerData) getProviderObject() (*models.CloudProvider, error) {

	request := new(services.GetCloudProviderRequest)
	request.ID = p.info.UUID

	provResp, err := p.client.GetCloudProvider(p.ctx, request)
	if err != nil {
		return nil, err
	}

	return provResp.GetCloudProvider(), nil

}

func (d *Data) newProvider(provider *models.CloudProvider) (*providerData, error) {

	prov := &providerData{
		parentCloud: d,
		info:        provider,
		apiServer: apiServer{
			client: d.cloud.APIServer,
			ctx:    d.cloud.ctx,
		},
	}

	provObj, err := prov.getProviderObject()
	prov.info = provObj

	if err != nil {
		return nil, err
	}

	return prov, nil
}

func (p *providerData) updateRegions() error {

	for _, region := range p.info.CloudRegions {
		newRegion, err := p.newRegion(region)
		if err != nil {
			return err
		}

		err = newRegion.updateVClouds()
		if err != nil {
			return err
		}

		p.regions = append(p.regions, newRegion)
	}
	return nil
}

func (d *Data) updateProviders() error {
	for _, provider := range d.info.CloudProviders {
		newProvider, err := d.newProvider(provider)
		if err != nil {
			return err
		}

		err = newProvider.updateRegions()
		if err != nil {
			return err
		}

		d.providers = append(d.providers, newProvider)
	}
	return nil
}

func getUserObject(ctx context.Context, uuid string,
	apiClient *client.HTTP) (*models.CloudUser, error) {

	request := new(services.GetCloudUserRequest)
	request.ID = uuid

	userResp, err := apiClient.GetCloudUser(ctx, request)
	if err != nil {
		return nil, err
	}
	return userResp.GetCloudUser(), nil

}

func (d *Data) updateUsers() error {
	for _, user := range d.info.CloudUserRefs {
		userObj, err := getUserObject(d.cloud.ctx, user.UUID, d.cloud.APIServer)
		if err != nil {
			return err
		}

		// Adding logic to handle a ssh key generation if not added as cred ref
		if userObj.CredentialRefs != nil {
			for _, cred := range userObj.CredentialRefs {
				credObj, err := getCredObject(d.cloud.ctx, d.cloud.APIServer, cred.UUID)
				if err != nil {
					return err
				}
				d.credentials = append(d.credentials, credObj)
			}
		}
		d.users = append(d.users, userObj)
	}
	return nil
}

func (i *instanceData) updateProtoModes(isDelRequest bool) error {
	for _, gwNodeRef := range i.info.ContrailMulticloudGWNodeBackRefs {
		gwNodeResp, err := i.client.GetContrailMulticloudGWNode(i.ctx,
			&services.GetContrailMulticloudGWNodeRequest{
				ID: gwNodeRef.UUID,
			},
		)
		if err != nil {
			return err
		}

		i.protocolsMode = gwNodeResp.GetContrailMulticloudGWNode().ProtocolsMode
		return nil
	}
	if isDelRequest {
		return nil
	}
	return errors.New("instance does not have a contrail-multicloud-gw-node ref")
}

func (i *instanceData) updateVrouterGW(role string, isDelRequest bool) error {

	switch role {
	case gatewayRole:
		return i.setMultiCloudGWNodeDefaultGW(isDelRequest)
	case computeRole:
		return i.setVrouterNodeDefaultGW(isDelRequest)
	}

	return fmt.Errorf("instance does not have a %s ref", role)

}

func (i *instanceData) setMultiCloudGWNodeDefaultGW(isDelRequest bool) error {

	for _, gwNodeRef := range i.info.ContrailMulticloudGWNodeBackRefs {
		response, err := i.client.GetContrailMulticloudGWNode(i.ctx,
			&services.GetContrailMulticloudGWNodeRequest{
				ID: gwNodeRef.UUID,
			},
		)
		if err != nil {
			return err
		}

		if response != nil {
			i.gateway = response.ContrailMulticloudGWNode.DefaultGateway
		}
		if i.gateway == "" && !isDelRequest {
			return fmt.Errorf(
				"default gateway is not set for contrail_multicloud_gw_node uuid: %s",
				gwNodeRef.UUID)
		}
		return nil
	}

	if isDelRequest {
		return nil
	}

	return fmt.Errorf(
		"contrailMulticloudGWNodeBackRefs are not present for instance: %s",
		i.info.UUID)
}

func (i *instanceData) setVrouterNodeDefaultGW(isDelRequest bool) error {

	for _, vrouterNodeRef := range i.info.ContrailVrouterNodeBackRefs {
		response, err := i.client.GetContrailVrouterNode(i.ctx,
			&services.GetContrailVrouterNodeRequest{
				ID: vrouterNodeRef.UUID,
			},
		)
		if err != nil {
			return err
		}

		if response != nil {
			vrouterNode := response.ContrailVrouterNode
			if vrouterNode.DefaultGateway != "" {
				i.gateway = vrouterNode.DefaultGateway
				return nil
			}
			response, err := i.client.GetContrailCluster(i.ctx,
				&services.GetContrailClusterRequest{
					ID: vrouterNode.ParentUUID,
				},
			)
			if err != nil {
				return err
			}
			i.gateway = response.ContrailCluster.DefaultGateway

			if i.gateway == "" && !isDelRequest {
				return fmt.Errorf(
					`default gateway is neither set for vrouter_node uuid: %s
					nor for contrail_cluster uuid: %s`,
					vrouterNodeRef.UUID, vrouterNode.ParentUUID)
			}
			return nil
		}
	}
	if isDelRequest {
		return nil
	}
	return fmt.Errorf(
		"contrailVrouterNodeBackRefs are not present for instance: %s",
		i.info.UUID)
}

func (i *instanceData) updatePvtIntf(isDelRequest bool) error {
	for _, port := range i.info.Ports {
		i.pvtIntf = port
		return nil
	}
	if isDelRequest {
		return nil
	}
	return fmt.Errorf("onprem node %s should have private ip address",
		i.info.Name)
}

func (i *instanceData) updateMCGWServices() error {
	for _, gwNodeRef := range i.info.ContrailMulticloudGWNodeBackRefs {
		gwNodeResp, err := i.client.GetContrailMulticloudGWNode(i.ctx,
			&services.GetContrailMulticloudGWNodeRequest{
				ID: gwNodeRef.UUID,
			},
		)
		if err != nil {
			return err
		}
		gwNode := gwNodeResp.GetContrailMulticloudGWNode()
		if gwNode.Services != nil {
			for _, v := range gwNode.Services {
				i.services = append(i.services, v)
			}
		}
	}
	return nil
}

func (c *Cloud) getCloudData(isDelRequest bool) (*Data, error) {

	cloudData, err := c.newCloudData()
	if err != nil {
		return nil, err
	}

	err = cloudData.update(isDelRequest)
	if err != nil {
		return nil, err
	}

	return cloudData, nil

}

func (c *Cloud) newCloudData() (*Data, error) {

	data := Data{}
	data.cloud = c

	cloudObject, err := GetCloud(c.ctx, c.APIServer, c.config.CloudID)
	if err != nil {
		return nil, err
	}

	data.info = cloudObject
	return &data, nil

}

func (d *Data) update(isDelRequest bool) error {

	d.delRequest = isDelRequest
	err := d.updateProviders()
	if err != nil {
		return err
	}

	err = d.updateUsers()
	if err != nil {
		return err
	}

	return nil
}

func (d *Data) isCloudCreated() bool {

	status := d.info.ProvisioningState
	if d.cloud.config.Action == createAction && (status == statusNoState || status == "") {
		return false
	}
	d.cloud.log.Infof("Cloud %s already provisioned, STATE: %s", d.info.UUID, status)
	return true
}

func (d *Data) isCloudUpdateRequest() bool {

	status := d.info.ProvisioningState
	if d.cloud.config.Action == updateAction && (status == statusNoState) {
		return true
	}
	return false
}

func (d *Data) isCloudPrivate() bool {

	for _, provider := range d.info.CloudProviders {
		if provider.Type == onPrem {
			return true
		}
	}
	return false
}

func (d *Data) isCloudPublic() bool {

	if !d.isCloudPrivate() {
		return true
	}
	return false
}

func (d *Data) hasProviderAWS() bool {

	for _, prov := range d.providers {
		if prov.info.Type == aws {
			return true
		}
	}
	return false

}

func (d *Data) hasProviderAzure() bool {
	for _, prov := range d.providers {
		if prov.info.Type == azure {
			return true
		}
	}
	return false
}

func (d *Data) getDefaultCloudUser() (*models.CloudUser, error) {
	for _, user := range d.users {
		return user, nil
	}
	return nil, errors.New("cloudUser ref not found with cloud object")
}

func (d *Data) getGatewayNodes() []*instanceData {
	gwNodes := []*instanceData{}
	for _, inst := range d.instances {
		if inst.info.ContrailMulticloudGWNodeBackRefs != nil {
			gwNodes = append(gwNodes, inst)
		}
	}
	return gwNodes
}
