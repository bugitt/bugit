// Copyright 2016 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package avatar

import (
	"testing"
)

func Test_RandomImage(t *testing.T) {
	_, err := RandomImage([]byte("gogs@local"))
	if err != nil {
		t.Fatal(err)
	}
}
