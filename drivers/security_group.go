package kubiqo

import (
	"context"

	v3 "github.com/exoscale/egoscale/v3"
)

func (d *Driver) createDefaultSecurityGroup(ctx context.Context, sgName string) (v3.UUID, error) {
	client, err := d.client(ctx)
	if err != nil {
		return "", err
	}

	op, err := client.CreateSecurityGroup(ctx, v3.CreateSecurityGroupRequest{
		Name:        sgName,
		Description: "created by rancher-machine",
	})
	if err != nil {
		return "", err
	}

	res, err := client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return "", err
	}

	sgID := res.Reference.ID
	sg := v3.SecurityGroupResource{
		ID:         sgID,
		Name:       sgName,
		Visibility: v3.SecurityGroupResourceVisibilityPrivate,
	}

	var (
		dockerPort         int64 = 2376
		swarmPort          int64 = 3376
		kubeApiPort        int64 = 6443
		httpPort           int64 = 80
		sshPort            int64 = 22
		rancherWebhookPort int64 = 8443
		httpsPort          int64 = 443
		supervisorPort     int64 = 9345
		nodeExporter       int64 = 9796

		etcdPorts            = []int64{2379, 2380}
		vxlanPorts           = []int64{4789, 4789}
		typhaPorts           = []int64{5473, 5473}
		flannelPorts         = []int64{8472, 8472}
		otherKubePorts       = []int64{10250, 10252}
		kubeProxyPorts       = []int64{10256, 10256}
		nodePorts            = []int64{30000, 32767}
		calicoPort     int64 = 179
	)

	publicRules := []v3.AddRuleToSecurityGroupRequest{
		{
			Description: "SSH",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   sshPort,
			EndPort:     sshPort,
		},
		{
			Description: "Docker",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   dockerPort,
			EndPort:     dockerPort,
		},
		{
			Description: "(Legacy) Standalone Swarm",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   swarmPort,
			EndPort:     swarmPort,
		},
		{
			Description: "Rancher webhook",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   rancherWebhookPort,
			EndPort:     rancherWebhookPort,
		},
		{
			Description: "Kubernetes API",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   kubeApiPort,
			EndPort:     kubeApiPort,
		},
		{
			Description: "HTTP",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   httpPort,
			EndPort:     httpPort,
		},
		{
			Description: "HTTPS",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   httpsPort,
			EndPort:     httpsPort,
		},
		{
			Description: "NodePort range (TCP)",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:   nodePorts[0],
			EndPort:     nodePorts[1],
		},
		{
			Description: "NodePort range (UDP)",
			Protocol:    v3.AddRuleToSecurityGroupRequestProtocolUDP,
			StartPort:   nodePorts[0],
			EndPort:     nodePorts[1],
		},
	}

	internalRules := []v3.AddRuleToSecurityGroupRequest{
		{
			Description:   "RKE2 supervisor API",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     supervisorPort,
			EndPort:       supervisorPort,
			SecurityGroup: &sg,
		},
		{
			Description:   "etcd client/peer",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     etcdPorts[0],
			EndPort:       etcdPorts[1],
			SecurityGroup: &sg,
		},
		{
			Description:   "Calico Typha",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     typhaPorts[0],
			EndPort:       typhaPorts[1],
			SecurityGroup: &sg,
		},
		{
			Description:   "kubelet / kube components",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     otherKubePorts[0],
			EndPort:       otherKubePorts[1],
			SecurityGroup: &sg,
		},
		{
			Description:   "kube-proxy",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     kubeProxyPorts[0],
			EndPort:       kubeProxyPorts[1],
			SecurityGroup: &sg,
		},
		{
			Description:   "Node exporter metrics",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     nodeExporter,
			EndPort:       nodeExporter,
			SecurityGroup: &sg,
		},
		{
			Description:   "Calico BGP",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolTCP,
			StartPort:     calicoPort,
			EndPort:       calicoPort,
			SecurityGroup: &sg,
		},
		{
			Description:   "Calico VXLAN",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolUDP,
			StartPort:     vxlanPorts[0],
			EndPort:       vxlanPorts[1],
			SecurityGroup: &sg,
		},
		{
			Description:   "Flannel VXLAN",
			Protocol:      v3.AddRuleToSecurityGroupRequestProtocolUDP,
			StartPort:     flannelPorts[0],
			EndPort:       flannelPorts[1],
			SecurityGroup: &sg,
		},
	}

	dualStack := []string{"0.0.0.0/0", "::/0"}
	for _, req := range publicRules {
		for _, sourceCidr := range dualStack {
			req.FlowDirection = v3.AddRuleToSecurityGroupRequestFlowDirectionIngress
			req.Network = sourceCidr
			if err := addRuleToSG(ctx, client, sgID, req); err != nil {
				return "", err
			}
		}
	}

	for _, req := range internalRules {
		req.FlowDirection = v3.AddRuleToSecurityGroupRequestFlowDirectionIngress
		if err := addRuleToSG(ctx, client, sgID, req); err != nil {
			return "", err
		}
	}

	return sgID, nil
}

func addRuleToSG(ctx context.Context, client *v3.Client, sgID v3.UUID, req v3.AddRuleToSecurityGroupRequest) error {
	op, err := client.AddRuleToSecurityGroup(ctx, sgID, req)
	if err != nil {
		return err
	}
	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	return err
}
