// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sqlstore

import (
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	sq "github.com/mattermost/squirrel"
)

const (
	HTV_ALIAS = "HT.Val"
)

type SqlHashTagStore struct {
	*SqlStore
}

// ==== query helper functions

func SQLLower(s string) string {
	return "LOWER(" + s + ")"
}

/**
 * Create a mysql/postgresql like statement,
 * normalizing the search query cases.
 **/
func HashTagValLike(s *string) sq.Like {
	return sq.Like{
		SQLLower(HTV_ALIAS): strings.ToLower(*s),
	}
}

/**
 * Filter the query by post author ID
 **/
func (s *SqlHashTagStore) OfUser(stmt sq.SelectBuilder, user *model.User) sq.SelectBuilder {
	return stmt.Where(sq.Eq{
		"post.user_id": user.Id,
	})
}

/**
 * Filter the query by channel type.
 **/
func (s *SqlHashTagStore) InChannelType(stmt sq.SelectBuilder, channel_type model.ChannelType) sq.SelectBuilder {
	return stmt.Where(sq.Eq{
		"channel.channel_type": channel_type,
	})
}

/// ===== CREATE NEW HASH_TAG

/**
 * Base function for saving a hash tag.
 **/
func (s *SqlHashTagStore) SaveHashTag(hash_tag *string, post_id *string) error {
	insert := sq.Insert("hashtag").
		Columns("val", "post_id").
		Values(strings.ToLower(*hash_tag), *post_id).
		Options("IGNORE")
	insert_q, args, err := insert.ToSql()
	if err != nil {
		return err
	}
	_, err2 := s.GetReplicaX().Exec(insert_q, args)
	if err != nil {
		return err2
	}
	return nil
}

/**
 * Convenience method for just saving hashtags from a post.
 **/
func (s *SqlHashTagStore) SaveHashTagFromPost(post *model.Post) error {
	hash_tags := strings.Split(post.Hashtags, " ")
	for _, hash_tag := range hash_tags {
		hash_tag = strings.Trim(hash_tag, " ")
		err := s.SaveHashTag(&hash_tag, &post.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

// === SELECTS

// ---- SELECT bases

/**
 * Base Select query from which all other select queries will be based off of.
 **/
func (s *SqlHashTagStore) HashTagQueryBase(count *uint64, columns []string) sq.SelectBuilder {
	return sq.Select(columns...).
		From("hash_tag HT").
		Join("post USING (post_id)").
		Join("channel USING (channel_id)").
		Limit(*count)
}

/**
 * return a base query with a filter `hash_tag LIKE 'query%'`
 **/
func (s *SqlHashTagStore) HashTagQueryBaseBegin(count *uint64, beginQuery *string, columns []string) sq.SelectBuilder {
	htq := model.LikeBegin(model.HashTagToString(beginQuery))
	return s.HashTagQueryBase(count, columns).
		Where(HashTagValLike(htq))
}

/**
 * return a base query with a filter `hash_tag LIKE '%query%'`
 **/
func (s *SqlHashTagStore) HashTagQueryBaseContains(count *uint64, beginQuery *string, columns []string) sq.SelectBuilder {
	htq := model.LikeContains(model.HashTagToString(beginQuery))
	return s.HashTagQueryBase(count, columns).
		Where(HashTagValLike(htq))
}

/**
 * Get a count result result form the database.
 **/
func (s *SqlHashTagStore) GetCount(queryBuilder *sq.SelectBuilder) ([]*model.HashTagCount, error) {
	result := []*model.HashTagCount{}
	s_query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	err = s.GetReplicaX().Get(result, s_query, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

/**
 * Get a timed result from the databae.
 **/
func (s *SqlHashTagStore) GetTimed(queryBuilder *sq.SelectBuilder) ([]*model.HashTagTimed, error) {
	result := []*model.HashTagTimed{}
	s_query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	err = s.GetReplicaX().Get(result, s_query, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ---- User hash tags when "#" typed

/**
 * Return a list of most-used hashtags by any user on the server.
 * Note we'll just be returning hashtags from *OPEN* channels.
 **/
func (s *SqlHashTagStore) GetMostUsedTags(user *model.User, count *uint64) ([]*model.HashTagCount, error) {
	columns := []string{HTV_ALIAS, "COUNT(post_id) AS post_count"}
	query := s.HashTagQueryBase(count, columns)
	query = s.OfUser(query, user).OrderBy("post_count")
	return s.GetCount(&query)
}

/**
 * Return a list of recent user hash tags, sorted by most recent to
 * least recent.
 **/
func (s *SqlHashTagStore) GetRecentUserHashTags(user *model.User, count *uint64) ([]*model.HashTagTimed, error) {
	columns := []string{HTV_ALIAS, "post.create_at AS when"}
	query := s.HashTagQueryBase(count, columns)
	query = s.OfUser(query, user).OrderBy("post_count")
	return s.GetTimed(&query)
}

// ---- User hash tags after "#"

/**
 * Return a list of user hash tags that begin with a certain query.
 * Return the stats sorted by `when`.
 **/
func (s *SqlHashTagStore) GetUserHashTagsBegin(user *model.User, hash_tag_query *string, count *uint64) ([]*model.HashTagTimed, error) {
	columns := []string{HTV_ALIAS, "post.create_at AS when"}
	query := s.HashTagQueryBaseBegin(count, hash_tag_query, columns)
	query = s.OfUser(query, user).OrderBy("post_count")
	return s.GetTimed(&query)
}

/**
 * Return a list of user hash tags that contain a certain query.
 * Return the stats sorted by `when`.
 **/
func (s *SqlHashTagStore) GetUserHashTagsContains(user *model.User, hash_tag_query *string, count *uint64) ([]*model.HashTagTimed, error) {
	columns := []string{HTV_ALIAS, "post.create_at AS when"}
	query := s.HashTagQueryBaseContains(count, hash_tag_query, columns)
	query = s.OfUser(query, user).OrderBy("post_count")
	return s.GetTimed(&query)
}

/**
 * a base function to get "unused" hashtags.
 * `QF` is a "query base" function.
 **/
func (s *SqlHashTagStore) GetUnusedBase(QF func(*uint64, *string, []string) sq.SelectBuilder, user *model.User, hash_tag_query *string, count *uint64) ([]*model.HashTagCount, error) {
	columns := []string{HTV_ALIAS, "COUNT(post) AS post_count"}
	inner_columns := []string{HTV_ALIAS}
	query := QF(count, hash_tag_query, columns).
		Where(sq.NotEq{
			HTV_ALIAS: s.OfUser(s.HashTagQueryBase(count, inner_columns), user),
		})
	return s.GetCount(&query)
}

/**
 * Return a list of hash tags that begin with a certain query the user has not used.
 * Return the stats sorted by `post_count`.
 **/
func (s *SqlHashTagStore) GetUnusedHashTagsBegin(user *model.User, hash_tag_query *string, count *uint64) ([]*model.HashTagCount, error) {
	return s.GetUnusedBase(s.HashTagQueryBaseBegin, user, hash_tag_query, count)
}

/**
 * Return a list of hash tags that contain with a certain query the user has not used.
 * Return the stats sorted by `post_count`.
 **/
func (s *SqlHashTagStore) GetUnusedHashTagsContain(user *model.User, hash_tag_query *string, count *uint64) ([]*model.HashTagCount, error) {
	return s.GetUnusedBase(s.HashTagQueryBaseContains, user, hash_tag_query, count)
}

// ---- MEGA-Selection function

func (s *SqlHashTagStore) QueryHashTagBoard(user *model.User, hash_tag_query *string, count *uint64) (*model.HashTagBoard, error) {
	board := model.HashTagBoard{}
	if len(*hash_tag_query) == 0 {
		// on "#"
		user_recent, err := s.GetRecentUserHashTags(user, count)
		board.UserRecent = user_recent
		if err != nil {
			return nil, err
		}
		board.ServerPopular, err = s.GetMostUsedTags(user, count)
		if err != nil {
			return nil, err
		}
	} else {
		// after "#"
		starting, err := s.GetUserHashTagsBegin(user, hash_tag_query, count)
		board.Starting = starting
		if err != nil {
			return nil, err
		}
		board.Containing, err = s.GetUserHashTagsContains(user, hash_tag_query, count)
		if err != nil {
			return nil, err
		}
		board.StartingUnused, err = s.GetUnusedHashTagsBegin(user, hash_tag_query, count)
		if err != nil {
			return nil, err
		}
		board.ContainingUnused, err = s.GetUnusedHashTagsContain(user, hash_tag_query, count)
		if err != nil {
			return nil, err
		}
	}
	return &board, nil
}
