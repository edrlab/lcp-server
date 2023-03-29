// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package sign

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

func Canon(in interface{}) ([]byte, error) {
	// the easiest way to canonicalize is to marshal it and reify it as a map
	// which will sort stuff correctly
	b, err := json.Marshal(in)
	if err != nil {
		return b, err
	}

	var jsonObj interface{} // map[string]interface{} ==> auto sorting

	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()
	for {
		if err := dec.Decode(&jsonObj); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// do not escape characters
	enc.SetEscapeHTML(false)
	err = enc.Encode(jsonObj)
	if err != nil {
		return nil, err
	}
	// remove the trailing newline, added by encode
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
