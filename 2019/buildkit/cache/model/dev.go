package model

const (
	// OktetoSyncVolume is the name of the volume used for files sync
	OktetoSyncVolume = "okteto-sync"

	// OktetoSyncContainer is the name of the container used for files sync
	OktetoSyncContainer = "okteto-syncthing"

	// this depends entirely on the container configuration
	syncthingMountPath = "/var/cnd-sync"
)

//Development represent the development settings of a container
type Development struct {
	Image      string   `json:"image,omitempty" yaml:"image,omitempty"`
	Command    string   `json:"command,omitempty" yaml:"command,omitempty"`
	Arguments  []string `json:"args,omitempty" yaml:"args,omitempty"`
	Path       string   `json:"path,omitempty" yaml:"path,omitempty"`
	Persistent bool     `json:"persistent,omitempty" yaml:"persistent,omitempty"`
}

// EnsureValues fills missing values in d with defaults
func (d *Development) EnsureValues(defaultImage string) {
	tailCommand := true
	if d.Image == "" {
		d.Image = defaultImage
		tailCommand = false
	}

	if d.Command == "" && tailCommand {
		d.Command = "tail"
		d.Arguments = []string{"-f", "/dev/null"}
	}

	if d.Path == "" {
		d.Path = "/src"
	}
}

// GetSyncthingContainer returns the syncthing container to append when on dev mode
func GetSyncthingContainer() *Container {
	return &Container{
		Image:  "okteto/syncthing:latest",
		Expose: []string{"8384", "22000"},
		Mounts: map[string]*Mount{OktetoSyncVolume: &Mount{Path: syncthingMountPath}},
	}
}
