// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

type HashTagPost struct {
	// Foreign Key to the HashTag object.
	HashTag string `json:"text"`
	// Foreign Key to the Post
	PostId string `json:"post_id"`
	// FK to Channel
	// This is to limit the number of lookups
	// May not use.
	ChannelId string `json:"channel_id"`
}
