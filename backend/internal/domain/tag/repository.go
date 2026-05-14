package tag

import "context"

// Repository defines the interface for tag storage.
type Repository interface {
	CreateTag(ctx context.Context, t *Tag) error
	GetTagByID(ctx context.Context, id uint) (*Tag, error)
	GetTagByName(ctx context.Context, name string) (*Tag, error)
	UpdateTag(ctx context.Context, t *Tag) error
	DeleteTag(ctx context.Context, id uint) error
	ListTags(ctx context.Context) ([]*Tag, error)

	AddTagToObject(ctx context.Context, to *TaggedObject) error
	RemoveTagFromObject(ctx context.Context, tagID, objectID uint, objectType string) error
	ListTagsForObject(ctx context.Context, objectID uint, objectType string) ([]*Tag, error)
	ListObjectsForTag(ctx context.Context, tagID uint, objectType string) ([]*TaggedObject, error)
}
