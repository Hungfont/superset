package embedded

import "context"

// Repository defines the interface for embedded and miscellaneous storage.
type Repository interface {
	CreateCssTemplate(ctx context.Context, t *CssTemplate) error
	GetCssTemplateByID(ctx context.Context, id uint) (*CssTemplate, error)
	UpdateCssTemplate(ctx context.Context, t *CssTemplate) error
	DeleteCssTemplate(ctx context.Context, id uint) error
	ListCssTemplates(ctx context.Context) ([]*CssTemplate, error)

	GetKeyValue(ctx context.Context, uuid string) (*KeyValue, error)
	SetKeyValue(ctx context.Context, kv *KeyValue) error
	DeleteKeyValue(ctx context.Context, uuid string) error

	CreateEmbeddedDashboard(ctx context.Context, ed *EmbeddedDashboard) error
	GetEmbeddedDashboardByUUID(ctx context.Context, uuid string) (*EmbeddedDashboard, error)
	GetEmbeddedDashboardByDashboardID(ctx context.Context, dashboardID uint) (*EmbeddedDashboard, error)
	UpdateEmbeddedDashboard(ctx context.Context, ed *EmbeddedDashboard) error
	DeleteEmbeddedDashboard(ctx context.Context, uuid string) error

	GetUserAttribute(ctx context.Context, userID uint) (*UserAttribute, error)
	SetUserAttribute(ctx context.Context, ua *UserAttribute) error
}
