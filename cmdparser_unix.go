// Copyright 2015-2016 Fredrik Lidström. All rights reserved.
// Use of this source code is governed by the standard MIT License (MIT)
// that can be found in the LICENSE file.

// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package cmdparser

import "os"

// UserHomeFolder is a simple cr0ss-platform function to retrieve the users home path using environment variables.
func UserHomeFolder() string {
	return os.Getenv("HOME")
}
