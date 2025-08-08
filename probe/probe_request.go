// SPDX-License-Identifier: LGPL-3.0-or-later

package probe

import (
	"fmt"
	"io"
	"net/http"
)

const url = "https://atlas.ripe.net/api/v2/probes/"

// Get probe information from API
func Get(id int) (*Probe, error) {
	c := &http.Client{}
	u := fmt.Sprintf("%s%d", url, id)

	resp, err := c.Get(u)

	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return FromJSON(body)
}
