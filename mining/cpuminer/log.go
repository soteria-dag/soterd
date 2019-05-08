// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cpuminer

import (
	"github.com/soteria-dag/soterd/soterlog"
)

// log is a logger that is initialized with no output filters.  This
// means the package will not perform any logging by default until the caller
// requests it.
var log soterlog.Logger

// The default amount of logging is none.
func init() {
	DisableLog()
}

// DisableLog disables all library log output.  Logging output is disabled
// by default until UseLogger is called.
func DisableLog() {
	log = soterlog.Disabled
}

// UseLogger uses a specified Logger to output package logging info.
func UseLogger(logger soterlog.Logger) {
	log = logger
}
