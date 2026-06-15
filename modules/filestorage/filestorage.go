package filestorage

import "github.com/openumi/umi"

func init() {
	umi.RegisterModule(FileStorage{})
}

type FileStorage struct {
	Root string `json:"root,omitempty"`
}

func (FileStorage) UniModule() umi.ModuleInfo {
	return umi.ModuleInfo{
		ID:  "umi.storage.file_system",
		New: func() umi.Module { return new(FileStorage) },
	}
}
