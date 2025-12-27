package kubiqo

import "os"

// Build a cloud-init user data string that will install and run
// docker.
func (d *Driver) getCloudInit() ([]byte, error) {
	var err error
	if d.UserDataFile != "" {
		d.UserData, err = os.ReadFile(d.UserDataFile)
	}

	return d.UserData, err
}
