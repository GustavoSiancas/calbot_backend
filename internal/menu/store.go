package menu

import (
	"context"

	"calbot/internal/domain"
)

type Store interface {
	ListActiveRootOptions(ctx context.Context) ([]domain.MenuOption, error)
	GetActiveOption(ctx context.Context, id int64) (domain.MenuOption, error)
	GetRandomActivePrompt(ctx context.Context, menuOptionID int64) (*domain.MenuOptionPrompt, error)
	ListActiveChildren(ctx context.Context, parentOptionID int64) ([]domain.MenuOption, error)
	GetWeightedRandomActiveResponseGroup(ctx context.Context, menuOptionID int64) (*domain.ResponseGroup, error)
	ListActiveResponseItems(ctx context.Context, responseGroupID int64) ([]domain.ResponseItem, error)
	CreateMenuOption(ctx context.Context, option domain.MenuOption) (domain.MenuOption, error)
	DeactivateMenuOption(ctx context.Context, id int64) error
	CreatePrompt(ctx context.Context, prompt domain.MenuOptionPrompt) (domain.MenuOptionPrompt, error)
	DeletePrompt(ctx context.Context, id int64) error
	SetPromptActive(ctx context.Context, id int64, isActive bool) (domain.MenuOptionPrompt, error)
	CreateResponseGroup(ctx context.Context, menuOptionID int64, group domain.ResponseGroup, items []domain.ResponseItem) (domain.ResponseGroup, []domain.ResponseItem, error)
	ListAllMenuOptions(ctx context.Context) ([]domain.MenuOption, error)
	ListAllPrompts(ctx context.Context, menuOptionID int64) ([]domain.MenuOptionPrompt, error)
	ListAllResponseGroups(ctx context.Context, menuOptionID int64) ([]domain.ResponseGroup, error)
	ListAllResponseItems(ctx context.Context, responseGroupID int64) ([]domain.ResponseItem, error)
	GetMenuOptionByID(ctx context.Context, id int64) (domain.MenuOption, error)
	ListMenuOptionTreeNodesByParent(ctx context.Context, parentOptionID *int64) ([]MenuOptionTreeNode, error)
	DeleteMenuOption(ctx context.Context, id int64) error
	HasResponseItems(ctx context.Context, menuOptionID int64) (bool, error)
	HasChildren(ctx context.Context, menuOptionID int64) (bool, error)
	DeleteResponseGroup(ctx context.Context, id int64) error
}

type MenuOptionTreeNode struct {
	MenuOption  domain.MenuOption
	HasResponse bool
	HasChildren bool
}
