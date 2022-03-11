// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

type HashTag struct {
	// The actual text used, minus the hash symbol e.g. `api`
	Text string `json:"text"`
	// Timestamp of the last usage of the HashTag.
	LastUsed int64 `json:"last_used"`
}

type HashTagWithStats struct {
	HashTag
	Count int64 `json:"count"`
}
