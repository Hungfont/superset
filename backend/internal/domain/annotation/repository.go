package annotation

import "context"

// Repository defines the interface for annotation storage.
type Repository interface {
	CreateLayer(ctx context.Context, layer *Layer) error
	GetLayerByID(ctx context.Context, id uint) (*Layer, error)
	UpdateLayer(ctx context.Context, layer *Layer) error
	DeleteLayer(ctx context.Context, id uint) error
	ListLayers(ctx context.Context) ([]*Layer, error)

	CreateAnnotation(ctx context.Context, annotation *Annotation) error
	GetAnnotationByID(ctx context.Context, id uint) (*Annotation, error)
	UpdateAnnotation(ctx context.Context, annotation *Annotation) error
	DeleteAnnotation(ctx context.Context, id uint) error
	ListAnnotationsByLayer(ctx context.Context, layerID uint) ([]*Annotation, error)
}
