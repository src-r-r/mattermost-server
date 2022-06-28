package model

import "time"

type HashTag struct {
	Val    string `json:"val"`
	PostId int    `json:"post_id"`
}

type HashTagCount struct {
	HashTag   *string
	PostCount int
}

type HashTagTimed struct {
	HashTag *string
	When    *time.Time
}

type HashTagBoard struct {
	// Determined before characters are typed
	UserRecent    []*HashTagTimed
	ServerPopular []*HashTagCount
	// Filled once characters are typed.
	Starting         []*HashTagTimed
	Containing       []*HashTagTimed
	StartingUnused   []*HashTagCount
	ContainingUnused []*HashTagCount
}
