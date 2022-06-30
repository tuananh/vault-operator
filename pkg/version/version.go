/*
Copyright 2022 Tuan Anh Tran <me@tuananh.org>
*/
package version

import (
	"github.com/acorn-io/baaah/pkg/version"
)

var (
	Tag = "v0.0.0-dev"
)

func Get() version.Version {
	return version.NewVersion(Tag)
}
