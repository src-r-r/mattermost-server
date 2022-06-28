// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api4

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestHashTag(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	api, err := Init(th.Server)
	require.NoError(t, err)
	session, _ := th.App.GetSession(th.Client.AuthToken)

	cli := th.CreateClient()
	_, _, err = cli.Login(th.BasicUser2.Username, th.BasicUser2.Password)
	require.NoError(t, err)

	messages := []string{
		"This is the first #hashta1",
		"Even this post will be returned because it #has the right letters",
		"But his #one will not #have the correct hashtags.",
		"#hashta3",
		"#hashta4",
		"#hashta5 and #hashta6",
		"#hashta7",
		"#hashta8",
		"#hashta9",
		"#hashta10",
		"#hashta11",
	}

	for _, message := range messages {
		handler := api.APIHandler(createPost)
		resp := httptest.NewRecorder()
		post := &model.Post{
			ChannelId: th.BasicChannel.Id,
			Message:   message,
		}

		postJSON, jsonErr := json.Marshal(post)
		require.NoError(t, jsonErr)
		req := httptest.NewRequest("POST", "/api/v4/posts", bytes.NewReader(postJSON))
		req.Header.Set(model.HeaderAuth, "Bearer "+session.Token)

		handler.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusCreated, resp.Code)
	}

	t.Run("Hash Tag suggestions are returned for valid hashtags", func(t *testing.T) {
		hash_tag := "has"

		th.App.UpdateConfig(func(cfg *model.Config) {
			cfg.ImageProxySettings.Enable = model.NewBool(false)
		})

		r, err := http.NewRequest("GET", th.Client.APIURL+"/hashtag/"+hash_tag, nil)
		require.NoError(t, err)
		r.Header.Set(model.HeaderAuth, th.Client.AuthType+" "+th.Client.AuthToken)

		resp, err := th.Client.HTTPClient.Do(r)
		require.NoError(t, err)

		hash_tags := []*string{}

		if jsonErr := json.NewDecoder(resp.Body).Decode(&hash_tags); jsonErr != nil {
			mlog.Warn("Failed to decode from JSON", mlog.Err(jsonErr))
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Len(t, hash_tags, 10)
	})

	t.Run("Setting a limit N returns up to N items", func(t *testing.T) {
		hash_tag := "has"
		limit := 5

		th.App.UpdateConfig(func(cfg *model.Config) {
			cfg.ImageProxySettings.Enable = model.NewBool(false)
		})

		r, err := http.NewRequest("GET", th.Client.APIURL+"/hashtag/"+hash_tag+"?limit="+strconv.Itoa(limit), nil)
		require.NoError(t, err)
		r.Header.Set(model.HeaderAuth, th.Client.AuthType+" "+th.Client.AuthToken)

		resp, err := th.Client.HTTPClient.Do(r)
		require.NoError(t, err)

		hash_tags := []*string{}

		if jsonErr := json.NewDecoder(resp.Body).Decode(&hash_tags); jsonErr != nil {
			mlog.Warn("Failed to decode from JSON", mlog.Err(jsonErr))
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Len(t, hash_tags, limit)

		// Make sure we are containing hashtags INSIDE the limit
		assert.NotContains(t, hash_tags, "hashta1")
		assert.NotContains(t, hash_tags, "has")
		assert.NotContains(t, hash_tags, "hashta3")
		assert.NotContains(t, hash_tags, "hashta4")
		assert.NotContains(t, hash_tags, "hashta5")

		// Make sure we aren't containing hashtags OUTSIDE the limit
		assert.NotContains(t, hash_tags, "hashta6")
		assert.NotContains(t, hash_tags, "hashta7")
		assert.NotContains(t, hash_tags, "hashta8")
		assert.NotContains(t, hash_tags, "hashta9")
		assert.NotContains(t, hash_tags, "hashta10")
		assert.NotContains(t, hash_tags, "hashta11")
	})
}
