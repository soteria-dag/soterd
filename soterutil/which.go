// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

import (
	"os"
	"path"
	"strings"
)

// GoBin returns the GOBIN path
func GoBin() string {
	goBin, exists := os.LookupEnv("GOBIN")
	if exists {
		return goBin
	}

	goPath, exists := os.LookupEnv("GOPATH")
	if exists {
		return path.Join(goPath, "bin")
	}

	home, exists := os.LookupEnv("HOME")
	if exists {
		return path.Join(home, "go", "bin")
	}

	return ""
}

// Which returns the path to a given file name, and a boolean of if it was found
// TODO(cedric): Support lookup on windows
func Which(name string) (string, bool) {
	_, err := os.Stat(name)
	if err == nil {
		return name, true
	}

	var defaultPaths = []string{
		"/bin",
		"/usr/bin",
		"/sbin",
		"/usr/sbin",
	}

	var paths []string
	p, exists := os.LookupEnv("PATH")
	if exists {
		paths = strings.Split(p, ":")
	} else {
		paths = defaultPaths
	}

	// We'll always search GOBIN and HOME/go/bin
	goBin := GoBin()
	if len(goBin) > 0 {
		paths = append(paths, goBin)
	}

	home, exists := os.LookupEnv("HOME")
	if exists {
		goBinAlt := path.Join(home, "go", "bin")
		if goBin != goBinAlt {
			paths = append(paths, goBinAlt)
		}
	}

	for _, p := range paths {
		tryPath := path.Join(p, name)
		_, err := os.Stat(tryPath)
		if err == nil {
			return tryPath, true
		}
	}

	return "", false
}