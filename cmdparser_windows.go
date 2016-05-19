// Copyright 2015-2016 Fredrik Lidström. All rights reserved.
// Use of this source code is governed by the standard MIT License (MIT)
// that can be found in the LICENSE file.

// +build windows

package cmdparser

import "os"

// UserHomeFolder is a simple cross-platform function to retrieve the users home path using environment variables.
func UserHomeFolder() string {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	if drive == "" || path == "" {
		return os.Getenv("USERPROFILE")
	} else {
		return drive + path
	}
}
