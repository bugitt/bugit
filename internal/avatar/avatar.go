// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package avatar

import (
	"image"

	"github.com/aofei/cameron"
)

const AVATAR_SIZE = 290

// RandomImage generates and returns a random avatar image unique to input data
// in default size (height and width).
func RandomImage(data []byte) (image.Image, error) {
	return cameron.Identicon(data, AVATAR_SIZE, 30), nil
}
