// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameFeedMode = "feed_modes"

// FeedMode mapped from table <feed_modes>
type FeedMode struct {
	ID       int32  `gorm:"column:id;primaryKey" json:"id"`
	FeedMode string `gorm:"column:feed_mode;not null" json:"feed_mode"`
}

// TableName FeedMode's table name
func (*FeedMode) TableName() string {
	return TableNameFeedMode
}