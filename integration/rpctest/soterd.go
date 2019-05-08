// Copyright (c) 2017 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpctest

import (
	"fmt"
	"go/build"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	// compileMtx guards access to the executable path so that the project is
	// only compiled once.
	compileMtx sync.Mutex

	// executablePath is the path to the compiled executable. This is the empty
	// string until soterd is compiled. This should not be accessed directly;
	// instead use the function soterdExecutablePath().
	executablePath string
)

// soterdExecutablePath returns a path to the soterd executable to be used by
// rpctests. To ensure the code tests against the most up-to-date version of
// soterd, this method compiles soterd the first time it is called. After that, the
// generated binary is used for subsequent test harnesses. The executable file
// is not cleaned up, but since it lives at a static path in a temp directory,
// it is not a big deal.
func soterdExecutablePath() (string, error) {
	compileMtx.Lock()
	defer compileMtx.Unlock()

	// If soterd has already been compiled, just use that.
	if len(executablePath) != 0 {
		return executablePath, nil
	}

	testDir, err := baseDir()
	if err != nil {
		return "", err
	}

	// Determine import path of this package. Not necessarily soteria-dag/soterd if
	// this is a forked repo.
	_, rpctestDir, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("Cannot get path to soterd source code")
	}
	soterdPkgPath := filepath.Join(rpctestDir, "..", "..", "..")
	soterdPkg, err := build.ImportDir(soterdPkgPath, build.FindOnly)
	if err != nil {
		return "", fmt.Errorf("Failed to build soterd: %v", err)
	}

	// The Package type returned by build.ImportDir() may have a ImportPath of "." when
	// the dir parameter is a package path for a `go mod` dependency (like this:
	// /home/me/go/pkg/mod/github.com/soteria-dag/soterd@v0.0.0-20190411003429-eb03f84f0e80
	// ).
	// If that's the case, we'll just use the relative path determined from the call to `runtime.Caller()`.
	//
	// This behaviour doesn't occur when a `replace` statement is used in the go.mod file, that has
	// dependencies resolved through the filesystem instead of a remote path like github.com. In that case
	// the ImportPath field would be the same as the dir parameter, like this:
	// /home/me/src/github.com/soteria-dag/soterd
	//
	var pkgPath string
	if soterdPkg.ImportPath == "." {
		pkgPath = soterdPkgPath
	} else {
		pkgPath = soterdPkg.ImportPath
	}

	// Build soterd and output an executable in a static temp path.
	outputPath := filepath.Join(testDir, "soterd")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", outputPath, pkgPath)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Failed to build soterd: %v", err)
	}

	// Save executable path so future calls do not recompile.
	executablePath = outputPath
	return executablePath, nil
}
