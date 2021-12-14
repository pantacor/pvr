package models

const (
	SourceTypeDocker = "docker"
	SourceTypePvr    = "pvr"
	SourceTypeRootFs = "rootfs"
)

type GetSTOptions struct {
	Force bool
}
