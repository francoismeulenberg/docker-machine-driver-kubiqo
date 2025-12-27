package kubiqo

import "strings"

// GetSSHHostname returns the hostname to use with SSH
func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

// GetSSHUsername returns the username to use with SSH
func (d *Driver) GetSSHUsername() string {
	if d.SSHUser == "" {
		name := strings.ToLower(d.Image)

		if strings.Contains(name, "ubuntu") {
			return "ubuntu"
		}
		if strings.Contains(name, "centos") {
			return "centos"
		}
		if strings.Contains(name, "redhat") {
			return "cloud-user"
		}
		if strings.Contains(name, "fedora") {
			return "fedora"
		}
		if strings.Contains(name, "coreos") {
			return "core"
		}
		if strings.Contains(name, "debian") {
			return "debian"
		}
		return defaultSSHUser
	}

	return d.SSHUser
}
