package menu

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"calbot/internal/domain"
	"calbot/internal/security"
)

var (
	ErrNotFound              = errors.New("menu option not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrInvalidInput          = errors.New("invalid menu option input")
	ErrMenuOptionHasResponse = errors.New("menu option already has response items")
	ErrMenuOptionHasChildren = errors.New("menu option already has child options")
)

type OptionDetail struct {
	MenuOption domain.MenuOption
	Prompt     *domain.MenuOptionPrompt
	Children   []domain.MenuOption
}

type ResponseGroupDetail struct {
	ResponseGroup domain.ResponseGroup
	ResponseItems []domain.ResponseItem
}

type StaffMenuOptionDetail struct {
	MenuOption     domain.MenuOption
	Prompts        []domain.MenuOptionPrompt
	ResponseGroups []ResponseGroupDetail
}

type Service struct {
	store Store
	jwt   *security.JWTService
}

func NewService(store Store, jwt *security.JWTService) *Service {
	return &Service{store: store, jwt: jwt}
}

type CreateInput struct {
	ParentOptionID *int64
	Title          string
	SortOrder      int
}

type PromptInput struct {
	Message string
	Weight  int
}

type ResponseGroupInput struct {
	Name        string
	Description *string
	Weight      int
	Items       []ResponseItemInput
}

type ResponseItemInput struct {
	Type      domain.ResponseItemType
	Text      *string
	URL       *string
	Caption   *string
	Metadata  json.RawMessage
	SortOrder int
}

func (s *Service) Create(ctx context.Context, staffToken string, input CreateInput) (domain.MenuOption, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOption{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.MenuOption{}, ErrInvalidInput
	}
	if input.ParentOptionID != nil {
		hasResponse, err := s.store.HasResponseItems(ctx, *input.ParentOptionID)
		if err != nil {
			return domain.MenuOption{}, err
		}
		if hasResponse {
			return domain.MenuOption{}, ErrMenuOptionHasResponse
		}
	}
	return s.store.CreateMenuOption(ctx, domain.MenuOption{ParentOptionID: input.ParentOptionID, Title: strings.TrimSpace(input.Title), SortOrder: input.SortOrder})
}

func (s *Service) Deactivate(ctx context.Context, staffToken string, id int64) error {
	if err := s.requireStaff(staffToken); err != nil {
		return err
	}
	if id < 1 {
		return ErrInvalidInput
	}
	if err := s.store.DeactivateMenuOption(ctx, id); err != nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, staffToken string, id int64) error {
	if err := s.requireStaff(staffToken); err != nil {
		return err
	}
	if id < 1 {
		return ErrInvalidInput
	}
	if err := s.store.DeleteMenuOption(ctx, id); err != nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) CreatePrompt(ctx context.Context, staffToken string, menuOptionID int64, input PromptInput) (domain.MenuOptionPrompt, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOptionPrompt{}, err
	}
	if menuOptionID < 1 || strings.TrimSpace(input.Message) == "" || input.Weight <= 0 {
		return domain.MenuOptionPrompt{}, ErrInvalidInput
	}
	return s.store.CreatePrompt(ctx, domain.MenuOptionPrompt{MenuOptionID: menuOptionID, Message: strings.TrimSpace(input.Message), Weight: input.Weight})
}

func (s *Service) DeletePrompt(ctx context.Context, staffToken string, id int64) error {
	if err := s.requireStaff(staffToken); err != nil {
		return err
	}
	if id < 1 {
		return ErrInvalidInput
	}
	if err := s.store.DeletePrompt(ctx, id); err != nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SetPromptActive(ctx context.Context, staffToken string, id int64, isActive bool) (domain.MenuOptionPrompt, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOptionPrompt{}, err
	}
	if id < 1 {
		return domain.MenuOptionPrompt{}, ErrInvalidInput
	}
	prompt, err := s.store.SetPromptActive(ctx, id, isActive)
	if err != nil {
		return domain.MenuOptionPrompt{}, ErrNotFound
	}
	return prompt, nil
}

func (s *Service) CreateResponseGroup(ctx context.Context, staffToken string, menuOptionID int64, input ResponseGroupInput) (domain.ResponseGroup, []domain.ResponseItem, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	if menuOptionID < 1 || strings.TrimSpace(input.Name) == "" || input.Weight <= 0 {
		return domain.ResponseGroup{}, nil, ErrInvalidInput
	}
	items := make([]domain.ResponseItem, 0, len(input.Items))
	for _, item := range input.Items {
		if !validResponseItem(item) {
			return domain.ResponseGroup{}, nil, ErrInvalidInput
		}
		items = append(items, domain.ResponseItem{Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	if len(items) > 0 {
		hasChildren, err := s.store.HasChildren(ctx, menuOptionID)
		if err != nil {
			return domain.ResponseGroup{}, nil, err
		}
		if hasChildren {
			return domain.ResponseGroup{}, nil, ErrMenuOptionHasChildren
		}
	}
	return s.store.CreateResponseGroup(ctx, menuOptionID, domain.ResponseGroup{Name: strings.TrimSpace(input.Name), Description: input.Description, Weight: input.Weight}, items)
}

func validResponseItem(item ResponseItemInput) bool {
	if len(item.Metadata) > 0 && !json.Valid(item.Metadata) {
		return false
	}
	switch item.Type {
	case domain.ResponseItemTypeText:
		return item.Text != nil && strings.TrimSpace(*item.Text) != ""
	case domain.ResponseItemTypeLink, domain.ResponseItemTypeImage, domain.ResponseItemTypeYouTube, domain.ResponseItemTypeVideo, domain.ResponseItemTypeDocument:
		return item.URL != nil && strings.TrimSpace(*item.URL) != ""
	default:
		return false
	}
}

func (s *Service) requireStaff(token string) error {
	claims, err := s.jwt.Validate(token)
	if err != nil {
		return ErrUnauthorized
	}
	if claims.Role != domain.UserRoleStaff {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ListRootOptions(ctx context.Context) ([]domain.MenuOption, error) {
	return s.store.ListActiveRootOptions(ctx)
}

func (s *Service) ListAllForStaff(ctx context.Context, staffToken string) ([]StaffMenuOptionDetail, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return nil, err
	}
	options, err := s.store.ListAllMenuOptions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]StaffMenuOptionDetail, 0, len(options))
	for _, option := range options {
		prompts, err := s.store.ListAllPrompts(ctx, option.ID)
		if err != nil {
			return nil, err
		}
		groups, err := s.store.ListAllResponseGroups(ctx, option.ID)
		if err != nil {
			return nil, err
		}
		groupDetails := make([]ResponseGroupDetail, 0, len(groups))
		for _, group := range groups {
			items, err := s.store.ListAllResponseItems(ctx, group.ID)
			if err != nil {
				return nil, err
			}
			groupDetails = append(groupDetails, ResponseGroupDetail{ResponseGroup: group, ResponseItems: items})
		}
		result = append(result, StaffMenuOptionDetail{MenuOption: option, Prompts: prompts, ResponseGroups: groupDetails})
	}
	return result, nil
}

func (s *Service) GetByIDForStaff(ctx context.Context, staffToken string, id int64) (domain.MenuOption, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOption{}, err
	}
	if id < 1 {
		return domain.MenuOption{}, ErrInvalidInput
	}
	option, err := s.store.GetMenuOptionByID(ctx, id)
	if err != nil {
		return domain.MenuOption{}, ErrNotFound
	}
	return option, nil
}

func (s *Service) ListByParentForStaff(ctx context.Context, staffToken string, parentOptionID *int64) ([]MenuOptionTreeNode, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return nil, err
	}
	if parentOptionID != nil && *parentOptionID < 1 {
		return nil, ErrInvalidInput
	}
	return s.store.ListMenuOptionTreeNodesByParent(ctx, parentOptionID)
}

func (s *Service) ListPromptsForStaff(ctx context.Context, staffToken string, menuOptionID int64) ([]domain.MenuOptionPrompt, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return nil, err
	}
	if menuOptionID < 1 {
		return nil, ErrInvalidInput
	}
	if _, err := s.store.GetMenuOptionByID(ctx, menuOptionID); err != nil {
		return nil, ErrNotFound
	}
	return s.store.ListAllPrompts(ctx, menuOptionID)
}

func (s *Service) ListResponseGroupsForStaff(ctx context.Context, staffToken string, menuOptionID int64) ([]ResponseGroupDetail, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return nil, err
	}
	if menuOptionID < 1 {
		return nil, ErrInvalidInput
	}
	if _, err := s.store.GetMenuOptionByID(ctx, menuOptionID); err != nil {
		return nil, ErrNotFound
	}
	groups, err := s.store.ListAllResponseGroups(ctx, menuOptionID)
	if err != nil {
		return nil, err
	}
	result := make([]ResponseGroupDetail, 0, len(groups))
	for _, group := range groups {
		items, err := s.store.ListAllResponseItems(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ResponseGroupDetail{ResponseGroup: group, ResponseItems: items})
	}
	return result, nil
}

func (s *Service) DeleteResponseGroup(ctx context.Context, staffToken string, id int64) error {
	if err := s.requireStaff(staffToken); err != nil {
		return err
	}
	if id < 1 {
		return ErrInvalidInput
	}
	if err := s.store.DeleteResponseGroup(ctx, id); err != nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) GetOptionDetail(ctx context.Context, id int64) (OptionDetail, error) {
	option, err := s.store.GetActiveOption(ctx, id)
	if err != nil {
		return OptionDetail{}, ErrNotFound
	}
	prompt, err := s.store.GetRandomActivePrompt(ctx, id)
	if err != nil {
		return OptionDetail{}, err
	}
	children, err := s.store.ListActiveChildren(ctx, id)
	if err != nil {
		return OptionDetail{}, err
	}
	return OptionDetail{MenuOption: option, Prompt: prompt, Children: children}, nil
}

func (s *Service) GetActiveResponse(ctx context.Context, menuOptionID int64) (*ResponseGroupDetail, error) {
	if _, err := s.store.GetActiveOption(ctx, menuOptionID); err != nil {
		return nil, ErrNotFound
	}
	group, err := s.store.GetWeightedRandomActiveResponseGroup(ctx, menuOptionID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}
	items, err := s.store.ListActiveResponseItems(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	return &ResponseGroupDetail{ResponseGroup: *group, ResponseItems: items}, nil
}
