//go:build gc && !wasm

/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package hack gives you some efficient functionality at the cost of
// breaking some Go rules.
package hack

import (
	"unsafe"
)

// String force casts a []byte to a string.
// USE AT YOUR OWN RISK
func String(b []byte) (s string) {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

// StringBytes returns the underlying bytes for a string. Modifying this byte slice
// will lead to undefined behavior.
func StringBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
