// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

import (
	"os"
	"path"
	"strings"
)

// Which returns the path to a given file name, and a boolean of if it was found
// TODO(cedric): Support lookup on windows
func Which(name string) (string, bool) {
	defaultPath := ":/bin:/usr/bin:/sbin:/usr/sbin"

	_, err := os.Stat(name)
	if err == nil {
		return name, true
	}

	var searchPath string
	val, exists := os.LookupEnv("PATH")
	if !exists {
		searchPath = defaultPath
	} else {
		searchPath = val
	}

	for _, p := range strings.Split(searchPath, ":") {
		tryPath := path.Join(p, name)
		_, err := os.Stat(tryPath)
		if err == nil {
			return tryPath, true
		}
	}

	return "", false
}