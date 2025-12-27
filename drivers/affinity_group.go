package kubiqo

import (
	"context"

	v3 "github.com/exoscale/egoscale/v3"
)

func (d *Driver) createDefaultAffinityGroup(ctx context.Context, agName string) (v3.UUID, error) {
	client, err := d.client(ctx)
	if err != nil {
		return "", err
	}

	resp, err := client.CreateAntiAffinityGroup(ctx, v3.CreateAntiAffinityGroupRequest{
		Name:        agName,
		Description: "created by rancher-machine",
	})
	if err != nil {
		return "", err
	}

	op, err := client.Wait(ctx, resp, v3.OperationStateSuccess)
	if err != nil {
		return "", err
	}

	return op.Reference.ID, nil
}
