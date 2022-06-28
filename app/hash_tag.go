// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"

	"github.com/mattermost/mattermost-server/v6/model"
)

/**
 * Returns the "hash tag board" that will be displayed in the autocompletion
 * menu.
 **/
func (a *App) QueryHashTag(user *model.User, hash_tag_query *string, count *uint64) (*model.HashTagBoard, *model.AppError) {
	hash_tags, err := a.Srv().GetStore().HashTag().QueryHashTagBoard(user, hash_tag_query, count)
	if err != nil {
		return nil, model.NewAppError("QueryHashTag", "app.hash_tag.query_hash_tag.app_error", nil, err.Error(), http.StatusBadRequest)
	}

	return hash_tags, nil
}
