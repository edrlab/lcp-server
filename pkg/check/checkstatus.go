// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"github.com/edrlab/lcp-server/pkg/lic"
)

// Check the license status document
func CheckStatusDoc(statusDoc lic.StatusDoc) error {

	// check that the status doc is valid vs the json schema

	// display the status of the license and the associated message

	// check that the document contains a link to the fresh license

	// check the mime-type of the link to the fresh license

	// check the present of a register link and its mime-type

	// check that it is templated and the url gets id and name params

	// check the presence of a renew link and its mime-type

	// check that it is templated and the url gets id and name params
	// or it is an html url and the target resource is accessible

	// check the presence of a return link and its mime-type

	// check that is is templated and the url gets id and name params

	// display the max end date of the license

	// indicate if events are present in the status document

	return nil
}
