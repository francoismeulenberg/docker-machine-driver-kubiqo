package kubiqo

import (
	"context"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
	"github.com/rancher/machine/libmachine/log"
)

func (d *Driver) client(ctx context.Context) (*v3.Client, error) {
	client, err := v3.NewClient(credentials.NewStaticCredentials(d.APIKey, d.APISecretKey))
	if err != nil {
		return nil, err
	}

	if d.URL != "" {
		client = client.WithEndpoint(v3.Endpoint(d.URL))
	}

	zones, err := client.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	zone, err := zones.FindZone(d.AvailabilityZone)
	if err != nil {
		return nil, err
	}

	log.Debugf("Availability zone %v = %s", d.AvailabilityZone, zone)
	client = client.WithEndpoint(zone.APIEndpoint)

	return client, nil
}

func (d *Driver) getInstance() (*v3.Instance, error) {
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		return nil, err
	}

	return client.GetInstance(ctx, d.ID)
}
