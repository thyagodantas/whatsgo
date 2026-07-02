// Copyright (c) 2024 thyagodantas
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WAVersionUpdateURL is the public endpoint used to discover the current
// WhatsApp web client version. The `version` query parameter is required
// by the server (any string works; the response is independent of it).
//
// Callers wanting a different endpoint (mirror, proxy, internal service)
// can simply skip this helper and pass the result of store.ParseVersion
// directly to store.SetWAVersion.
const WAVersionUpdateURL = "https://web.whatsapp.com/check-update?version=0&platform=web"

// FetchLatestWAVersion queries the WhatsApp web check-update endpoint and
// returns the current client version as a WAVersionContainer.
//
// This is a helper, not a side-effect: the returned version is not applied
// to the library automatically. The caller decides what to do with it.
//
// Typical usage:
//
//	version, err := store.FetchLatestWAVersion(context.Background(), http.DefaultClient)
//	if err != nil {
//	    log.Warnf("could not fetch latest WA version, using bundled fallback: %v", err)
//	} else {
//	    store.SetWAVersion(version)
//	}
//
// The bundled fallback in BaseClientPayload (see clientpayload.go) is used
// when the caller does not invoke this function. Keeping the default
// predictable avoids surprising failures when this endpoint is unreachable
// (offline environments, captive portals, blocking, etc).
//
// The function does no caching of its own. If you want to avoid hitting the
// endpoint on every call, cache the result on the caller side.
func FetchLatestWAVersion(ctx context.Context, httpClient *http.Client) (WAVersionContainer, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, WAVersionUpdateURL, nil)
	if err != nil {
		return WAVersionContainer{}, fmt.Errorf("waversion: build request: %w", err)
	}
	req.Header.Set("User-Agent", "whatsgo")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return WAVersionContainer{}, fmt.Errorf("waversion: request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return WAVersionContainer{}, fmt.Errorf("waversion: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		CurrentVersion string `json:"currentVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return WAVersionContainer{}, fmt.Errorf("waversion: decode response: %w", err)
	}
	if payload.CurrentVersion == "" {
		return WAVersionContainer{}, fmt.Errorf("waversion: response missing currentVersion")
	}
	version, err := ParseVersion(payload.CurrentVersion)
	if err != nil {
		return WAVersionContainer{}, fmt.Errorf("waversion: parse currentVersion %q: %w", payload.CurrentVersion, err)
	}
	return version, nil
}