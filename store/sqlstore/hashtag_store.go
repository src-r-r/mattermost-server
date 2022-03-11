// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/dsnet/compress/internal/errors"
	"github.com/mattermost/mattermost-server/v6/einterfaces"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/store"
	"github.com/mattermost/mattermost-server/v6/utils"
)

type SqlHashTagStore struct {
	*SqlStore
	metrics einterfaces.MetricsInterface
}

func hashTagSliceColumnsWithTypes() []struct {
	Name string
	Type reflect.Kind
} {
	return []struct {
		Name string
		Type reflect.Kind
	}{
		{"Text", reflect.String},
		{"LastUsed", reflect.Int64},
	}
}

func newSqlHashTagStore(sqlStore *SqlStore, metrics einterfaces.MetricsInterface) store.HashTagStore {
	return &SqlHashTagStore{
		SqlStore: sqlStore,
		metrics:  metrics,
	}
}

func (s *SqlHashTagStore) Get(tagText string) (*model.HashTag, error) {
	db := s.GetReplicaX()

	query := sq.Select("HashTag.*").
		From("HashTag").
		Where(sq.Eq{"HashTag.Text": tagText})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	hashTag := model.HashTag{}

	err = db.Get(&hashTag, sql, args...)
	if err != nil {
		return nil, err
	}
	return &hashTag, nil
}

func isPostPublic(s *SqlHashTagStore, post *model.Post) bool {
	ch_st := s.SqlStore.Channel()

	// First ensure post belongs to a channel and is a public post
	post_channel, _ := ch_st.Get(post.ChannelId, false)
	if post_channel == nil {
		return false
	}

	if post_channel.Type != model.ChannelTypeOpen {
		return false
	}

}

/**
 * Get a slice of hashtags associated with a *public* post.
 * If the post is not public, this will produce an error.
 *
 * @param post The post for the hashtags
 *
 * @return
 *    - slice of raw hashtags (e.g. {"remote", "server", ...})
 *    - Slice of HashTag objects for the post
 *    - error, if given.
 **/
func (s *SqlHashTagStore) GetPostHashTags(post *model.Post) ([]string, *model.HashTagList, error) {
	db := s.GetReplicaX()

	query := sq.Select("HashTag.*").
		From("HashTag").
		Join("HashTagPost ON HashTag.text=HashTagPost.text").
		Where(sq.Eq{"HashTagPost.PostId": post.Id})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, nil, err
	}

	hashTags := model.HashTagList{}
	rawTagList := []string{}

	err = db.Select(&hashTags, sql, args...)
	if err != nil {
		return nil, nil, err
	}

	for _, hashTag := range hashTags {
		rawTagList = append(rawTagList, hashTag.Text)
	}

	return rawTagList, &hashTags, nil
}

/**
 * Either update or add hashtags. Return the created hashtags.
 **/
func (s *SqlHashTagStore) UpdateMultiple(tagTexts []*string, post *model.Post) (*model.HashTagList, error) {

	db := s.GetReplicaX()
	ch_st := s.SqlStore.Channel()

	// First ensure post belongs to a channel and is a public post
	post_channel, err := ch_st.Get(post.ChannelId, false)
	if post_channel == nil {
		return nil, errors.Wrapf(err, "failed getting channels with channelId=%s", post.ChannelId)
	}

	if post_channel.Type != model.ChannelTypeOpen {
		return nil, errors.Wrapf(err, "post is not public, with PostId=%s", post.Id)
	}

	// Compose the 3 queries in tandem

	q_update := sq.Update("HashTag")
	q_insert := sq.Insert("HashTag")
	q_htpost := sq.Insert("HashTagPost")

	raw_tags, post_existing, err := s.GetPostHashTags(post)
	_ = post_existing
	if err != nil {
		return nil, errors.Wrapf(err, "UpdateMultiple")
	}

	for _, tagText := range tagTexts {
		existing, _ := s.Get(*tagText)
		// If the hashtag doesn't exist at all, create it
		if existing == nil {
			q_insert.SetMap(sq.Eq{
				"text": tagText,
			})
		}
		// If the hashtag doesn't exist in the post, add it to the "insert" list
		if !utils.StringInSlice(*tagText, raw_tags) {
			q_htpost.SetMap(sq.Eq{
				"text":    tagText,
				"post_id": post.Id,
			})
		}
		// Assume *everything* is an update
		q_update = q_update.Set("lastUsed", post.EditAt)
	}

	// Execute the composed queries

	// INSERT hashtag
	sql, args, err := q_insert.ToSql()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	// INSERT hashtag_post
	sql, args, err = q_htpost.ToSql()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	// UPDATE hashtag
	sql, args, err = q_update.ToSql()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(sql, args...)
	if err != nil {
		return nil, err
	}

	resultList := model.HashTagList{}

	return &resultList, nil
}
