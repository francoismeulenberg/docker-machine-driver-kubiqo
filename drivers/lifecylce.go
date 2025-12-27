package kubiqo

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/rancher/machine/libmachine/drivers"
	"github.com/rancher/machine/libmachine/log"
	"github.com/rancher/machine/libmachine/mcnutils"
	"github.com/rancher/machine/libmachine/ssh"
	"github.com/rancher/machine/libmachine/state"
)

// GetURL returns a Docker compatible host URL for connecting to this host
// e.g tcp://10.1.2.3:2376
func (d *Driver) GetURL() (string, error) {
	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}

	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, "2376")), nil
}

// GetState returns a github.com/machine/libmachine/state.State representing the state of the host (running, stopped, etc.)
func (d *Driver) GetState() (state.State, error) {
	instance, err := d.getInstance()
	if err != nil {
		return state.Error, err
	}
	switch instance.State {
	case v3.InstanceStateStarting:
		return state.Starting, nil
	case v3.InstanceStateRunning:
		return state.Running, nil
	case v3.InstanceStateStopping:
		return state.Stopping, nil
	case v3.InstanceStateStopped:
		return state.Stopped, nil
	case v3.InstanceStateDestroying:
		return state.Stopped, nil
	case v3.InstanceStateDestroyed:
		return state.Stopped, nil
	case v3.InstanceStateExpunging:
		return state.Stopped, nil
	case v3.InstanceStateMigrating:
		return state.Paused, nil
	case v3.InstanceStateError:
		return state.Error, nil
	}
	return state.None, nil
}

// Create creates the Instance acting as the docker host
func (d *Driver) Create() error {
	cloudInit, err := d.getCloudInit()
	if err != nil {
		return err
	}

	ctx := context.Background()
	log.Infof("Querying exoscale for the requested parameters...")
	client, err := d.client(ctx)
	if err != nil {
		return err
	}

	// Image
	templates, err := client.ListTemplates(ctx)
	if err != nil {
		return err
	}

	template := v3.Template{}

	image := strings.ToLower(d.Image)
	re := regexp.MustCompile(`^Linux (?P<name>.+?) (?P<version>[0-9.]+)\b`)

	for _, tpl := range templates.Templates {
		// Keep only 10GiB images
		if tpl.Size>>30 != 10 {
			continue
		}

		fullname := strings.ToLower(tpl.Name)
		if image == fullname {
			template = tpl
			break
		}

		submatch := re.FindStringSubmatch(tpl.Name)
		if len(submatch) > 0 {
			name := strings.ReplaceAll(strings.ToLower(submatch[1]), " ", "-")
			version := submatch[2]
			shortname := fmt.Sprintf("%s-%s", name, version)

			if image == shortname {
				template = tpl
				break
			}
		}
	}
	if template.ID == "" {
		return fmt.Errorf("unable to find image %v", d.Image)
	}

	// Reading the username from the template
	if template.DefaultUser != "" {
		d.SSHUser = template.DefaultUser
	}
	log.Debugf("Image %v(10) = %s (%s)", d.Image, template.ID, d.SSHUser)

	// Profile UUID
	instTypes, err := client.ListInstanceTypes(ctx)
	if err != nil {
		return err
	}

	instType, err := instTypes.FindInstanceTypeByIdOrFamilyAndSize(d.InstanceProfile)
	if err != nil {
		return err
	}

	log.Debugf("Profile %v = %v", d.InstanceProfile, instType)

	// Security groups
	sgs := make([]v3.SecurityGroup, 0, len(d.SecurityGroups))
	for _, sgName := range d.SecurityGroups {
		if sgName == "" {
			continue
		}

		sglist, err := client.ListSecurityGroups(ctx)
		if err != nil {
			return err
		}

		sg, err := sglist.FindSecurityGroup(sgName)
		if err != nil && !errors.Is(err, v3.ErrNotFound) {
			return err
		}

		var sgID v3.UUID
		if errors.Is(err, v3.ErrNotFound) {
			log.Infof("Security group %v does not exist. Creating it...", sgName)
			newSGID, err := d.createDefaultSecurityGroup(ctx, sgName)
			if err != nil {
				return err
			}

			sgID = newSGID
		} else {
			sgID = sg.ID
		}

		log.Debugf("Security group %v = %s", sgName, sgID)
		sgs = append(sgs, v3.SecurityGroup{
			ID: sgID,
		})
	}

	// Affinity Groups
	var ags []v3.AntiAffinityGroup
	for _, group := range d.AffinityGroups {
		if group == "" {
			continue
		}

		agList, err := client.ListAntiAffinityGroups(ctx)
		if err != nil {
			return err
		}

		ag, err := agList.FindAntiAffinityGroup(group)
		if err != nil && !errors.Is(err, v3.ErrNotFound) {
			return err
		}

		var agID v3.UUID
		if errors.Is(err, v3.ErrNotFound) {
			log.Infof("Affinity Group %v does not exist, create it", group)
			newAGID, err := d.createDefaultAffinityGroup(ctx, group)
			if err != nil {
				return err
			}
			agID = newAGID
		} else {
			agID = ag.ID
		}

		log.Debugf("Affinity group %v = %s", group, agID)
		ags = append(ags, v3.AntiAffinityGroup{
			ID: agID,
		})
	}

	// SSH key pair
	if d.SSHKey == "" {
		keyPairName := fmt.Sprintf("rancher-machine-%s", d.MachineName)
		log.Infof("Generate an SSH keypair...")

		err = ssh.GenerateSSHKey(d.GetSSHKeyPath())
		if err != nil {
			return err
		}

		pubKeyPath := d.ResolveStorePath("id_rsa.pub")
		pubKey, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return err
		}

		op, err := client.RegisterSSHKey(ctx, v3.RegisterSSHKeyRequest{
			Name:      keyPairName,
			PublicKey: string(pubKey),
		})
		if err != nil {
			return fmt.Errorf("SSH Key pair creation failed %s", err)
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return fmt.Errorf("SSH Key pair creation failed %s", err)
		}

		d.KeyPair = keyPairName
	} else {
		log.Infof("Importing SSH key from %s", d.SSHKey)

		sshKey := d.SSHKey
		if strings.HasPrefix(sshKey, "~/") {
			usr, _ := user.Current()
			sshKey = filepath.Join(usr.HomeDir, sshKey[2:])
		} else {
			var errA error
			if sshKey, errA = filepath.Abs(sshKey); errA != nil {
				return errA
			}
		}

		// Sending the SSH public key through the cloud-init config
		pubKey, errR := os.ReadFile(sshKey + ".pub")
		if errR != nil {
			return fmt.Errorf("cannot read SSH public key %s", errR)
		}

		sshAuthorizedKeys := `
ssh_authorized_keys:
- `
		cloudInit = bytes.Join([][]byte{cloudInit, []byte(sshAuthorizedKeys), pubKey}, []byte(""))

		// Copying the private key into rancher-machine
		if errCopy := mcnutils.CopyFile(sshKey, d.GetSSHKeyPath()); errCopy != nil {
			return fmt.Errorf("unable to copy SSH file: %s", errCopy)
		}
		if errChmod := os.Chmod(d.GetSSHKeyPath(), 0600); errChmod != nil {
			return fmt.Errorf("unable to set permissions on the SSH file: %s", errChmod)
		}
	}

	sshKey, err := client.GetSSHKey(ctx, d.KeyPair)
	if err != nil {
		return err
	}

	log.Infof("Spawn exoscale host...")
	log.Debugf("Using the following cloud-init file:")
	log.Debugf("%s", string(cloudInit))

	// Base64 encode the userdata
	d.UserData = cloudInit
	encodedUserData := base64.StdEncoding.EncodeToString(d.UserData)

	op, err := client.CreateInstance(ctx, v3.CreateInstanceRequest{
		Template:           &template,
		Ipv6Enabled:        v3.Bool(true),
		DiskSize:           d.DiskSize,
		InstanceType:       &instType,
		UserData:           encodedUserData,
		Name:               d.MachineName,
		SSHKeys:            []v3.SSHKey{*sshKey},
		SecurityGroups:     sgs,
		AntiAffinityGroups: ags,
	})
	if err != nil {
		return err
	}

	log.Infof("Deploying %s...", d.MachineName)

	res, err := client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		return err
	}

	instance, err := client.GetInstance(ctx, res.Reference.ID)
	if err != nil {
		return err
	}

	IPAddress := instance.PublicIP.String()
	if IPAddress != "<nil>" {
		d.IPAddress = IPAddress
	}
	d.ID = instance.ID
	log.Infof("IP Address: %v, SSH User: %v", d.IPAddress, d.GetSSHUsername())

	if instance.Template != nil && instance.Template.PasswordEnabled != nil && *instance.Template.PasswordEnabled {
		res, err := client.RevealInstancePassword(ctx, instance.ID)
		if err != nil {
			return err
		}

		d.Password = res.Password
	}

	// Destroy the SSH key
	if d.KeyPair != "" {
		if err := drivers.WaitForSSH(d); err != nil {
			return err
		}

		op, err := client.DeleteSSHKey(ctx, d.KeyPair)
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return err
		}

		d.KeyPair = ""
	}

	return nil
}

// Start starts the existing Instance.
func (d *Driver) Start() error {
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		return err
	}

	op, err := client.StartInstance(ctx, d.ID, v3.StartInstanceRequest{})
	if err != nil {
		return err
	}

	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)

	return err
}

// Stop stops the existing Instance.
func (d *Driver) Stop() error {
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		return err
	}

	op, err := client.StopInstance(ctx, d.ID)
	if err != nil {
		return err
	}

	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)

	return err
}

// Restart reboots the existing Instance.
func (d *Driver) Restart() error {
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		return err
	}

	op, err := client.RebootInstance(ctx, d.ID)
	if err != nil {
		return err
	}
	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)

	return err
}

// Kill stops a host forcefully (same as Stop)
func (d *Driver) Kill() error {
	return d.Stop()
}

// Remove destroys the Instance and the associated SSH key.
func (d *Driver) Remove() error {
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		return err
	}

	// Destroy the SSH key
	if d.KeyPair != "" {
		op, err := client.DeleteSSHKey(ctx, d.KeyPair)
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return err
		}
	}

	// Destroy the Instance
	if d.ID != "" {
		op, err := client.DeleteInstance(ctx, d.ID)
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return err
		}
	}

	//TODO: cleanup Anti-Affinities and Security-Groups, not urgent for now.
	log.Infof("The Anti-Affinity group and Security group were not removed")

	return nil
}
