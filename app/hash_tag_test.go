// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/maxatome/go-testdeep/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashTagSearch(t *testing.T) {

	// Create a test time for January 1st, 2016. Base it off hour.
	test_date := func(hour int) *time.Time {
		dt := time.Date(2016, time.January, 1, hour, 0, 0, 0, time.Local)
		return &dt
	}
	// the prrevious function, but with the unix timestamp
	test_date_x := func(hour int) int64 {
		return (*test_date(hour)).Unix()
	}

	posts := []*model.Post{}

	count := uint64(10)

	// The hashtags to expect. Store them as variables so that they
	// can be references.
	ht_tag1 := "tag1"
	ht_tag2 := "tag2"
	ht_taguser3 := "taguser3"
	// ht_usertag4 := "usertag4"
	ht_usertagity := "usertagity"
	ht_tagstartunused := "tagstartunused"
	ht_containingtagunused := "containingtagunused"

	// Expected result on an empty string (when they type "#")
	expected_onhash := model.HashTagBoard{
		ServerPopular: []*model.HashTagCount{
			{
				HashTag:   &ht_tag1,
				PostCount: 2,
			},
			{
				HashTag:   &ht_tag2,
				PostCount: 2,
			},
		},
		UserRecent: []*model.HashTagTimed{
			{
				HashTag: &ht_taguser3,
				When:    test_date(3),
			},
		},
	}

	// The expected result when a user types in something
	expected_ontext := model.HashTagBoard{
		Starting: []*model.HashTagTimed{
			{
				HashTag: &ht_tag1,
				When:    test_date(3),
			},
			{
				HashTag: &ht_tag2,
				When:    test_date(3),
			},
			{
				HashTag: &ht_taguser3,
				When:    test_date(3),
			},
		},
		Containing: []*model.HashTagTimed{
			{
				HashTag: &ht_usertagity,
				When:    test_date(4),
			},
			{
				HashTag: &ht_usertagity,
				When:    test_date(3),
			},
		},
		StartingUnused: []*model.HashTagCount{
			{
				HashTag:   &ht_tagstartunused,
				PostCount: 2,
			},
		},
		ContainingUnused: []*model.HashTagCount{
			{
				HashTag:   &ht_containingtagunused,
				PostCount: 1,
			},
		},
	}

	// The expected result when a user types in an invalid hashtag
	expected_oninvalid := model.HashTagBoard{}

	// Create some posts for the users.

	setup := func(t *testing.T, enableElasticsearch bool) (*TestHelper, []*model.Post) {
		th := Setup(t).InitBasic()

		// The "reference frame" will be th.BasicUser, who's part of
		// th.BasicChannel

		// Private posts; no hashtags should be included
		post, err := th.App.CreatePost(th.Context, &model.Post{
			UserId:    th.PrivateTeamUser.Id,
			ChannelId: th.PrivateChannel.Id,
			Message:   "#tag1, #tag2 #taguser3",
			CreateAt:  test_date_x(0),
		}, th.BasicChannel, false, true)
		require.Nil(t, err)
		posts = append(posts, post)

		// Public posts

		// Hashtags by th.PrivateTeamUser
		post, err = th.App.CreatePost(th.Context, &model.Post{
			UserId:    th.PrivateTeamUser.Id,
			ChannelId: th.BasicChannel.Id,
			Message:   "a message #tag1 #tagstartunused",
			CreateAt:  test_date_x(1),
		}, th.BasicChannel, false, true)
		require.Nil(t, err)
		posts = append(posts, post)

		post, err = th.App.CreatePost(th.Context, &model.Post{
			UserId:    th.PrivateTeamUser.Id,
			ChannelId: th.BasicChannel.Id,
			Message:   "a message #tag1 #tag2 #tagstartunused #continingtagunused",
			CreateAt:  test_date_x(2),
		}, th.BasicChannel, false, true)
		require.Nil(t, err)
		posts = append(posts, post)

		// Hashtags by th.BasicUser
		post, err = th.App.CreatePost(th.Context, &model.Post{
			UserId:    th.BasicUser.Id,
			ChannelId: th.BasicChannel.Id,
			Message:   "a message #tag1 #tag2 #taguser3",
			CreateAt:  test_date_x(3),
		}, th.BasicChannel, false, true)

		require.Nil(t, err)
		posts = append(posts, post)

		post, err = th.App.CreatePost(th.Context, &model.Post{
			UserId:    th.BasicUser.Id,
			ChannelId: th.BasicChannel.Id,
			Message:   "a message #usertagity",
			CreateAt:  test_date_x(4),
		}, th.BasicChannel, false, true)

		require.Nil(t, err)
		posts = append(posts, post)

		return th, posts
	}

	// Run the tests

	t.Run("An empty string returns UserRecent, ServerPopular", func(t *testing.T) {
		th, posts := setup(t, false)
		_ = posts
		defer th.TearDown()

		hash_tag_query := ""

		results, err := th.App.QueryHashTag(th.BasicUser, &hash_tag_query, &count)
		_ = err

		td.Cmp(t, results, &expected_onhash)

		assert.Nil(t, err)
	})
	t.Run("A string with 'tag' returns Starting, Containing, StartingUnused, ContainingUnused", func(t *testing.T) {
		th, posts := setup(t, false)
		_ = posts
		defer th.TearDown()

		hash_tag_query := "tag"

		results, err := th.App.QueryHashTag(th.BasicUser, &hash_tag_query, &count)
		_ = err

		td.Cmp(t, results, &expected_ontext)

		assert.Nil(t, err)
	})
	t.Run("A starting string with 'iehihighie' returns nothing", func(t *testing.T) {
		th, posts := setup(t, false)
		_ = posts
		defer th.TearDown()

		hash_tag_query := "iehihighie"

		results, err := th.App.QueryHashTag(th.BasicUser, &hash_tag_query, &count)
		_ = err

		td.Cmp(t, results, &expected_oninvalid)

		assert.Nil(t, err)
	})
}
