// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"

	httputil "github.com/pingcap/tidb-operator/v2/pkg/utils/http"
)

// StoreLimitType identifies a per-store incoming scheduling rate.
type StoreLimitType string

const (
	StoreLimitAddPeer          StoreLimitType = "add-peer"
	StoreLimitTransferLeaderIn StoreLimitType = "transfer-leader-in"
)

// StoreLimits uses pointers to distinguish unsupported fields from zero rates.
type StoreLimits struct {
	AddPeer          *float64 `json:"add-peer"`
	TransferLeaderIn *float64 `json:"transfer-leader-in"`
}

func (c *pdClient) GetStoreLimits(ctx context.Context) (map[string]StoreLimits, error) {
	body, err := httputil.GetBodyOK(ctx, c.httpClient, fmt.Sprintf("%s/%s/limit", c.url, storesPrefix))
	if err != nil {
		return nil, err
	}
	var limits map[string]StoreLimits
	if err := json.Unmarshal(body, &limits); err != nil {
		return nil, err
	}
	return limits, nil
}

func (c *pdClient) SetStoreLimit(ctx context.Context, storeID string, typ StoreLimitType, rate float64) error {
	if typ != StoreLimitAddPeer && typ != StoreLimitTransferLeaderIn {
		return fmt.Errorf("unsupported incoming store limit %q", typ)
	}
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return fmt.Errorf("store limit must be a positive finite rate")
	}
	body, err := json.Marshal(struct {
		Type StoreLimitType `json:"type"`
		Rate float64        `json:"rate"`
	}{typ, rate})
	if err != nil {
		return err
	}
	apiURL := fmt.Sprintf("%s/%s/%s/limit", c.url, storePrefix, url.PathEscape(storeID))
	_, err = httputil.PostBodyOK(ctx, c.httpClient, apiURL, bytes.NewReader(body))
	return err
}
