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
	ErrNotFound                           = errors.New("menu option not found")
	ErrUnauthorized                       = errors.New("unauthorized")
	ErrForbidden                          = errors.New("forbidden")
	ErrInvalidInput                       = errors.New("invalid menu option input")
	ErrResponseGroupDoesNotBelongToOption = errors.New("response group does not belong to menu option")
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
	Description    *string
	SortOrder      int
	Prompts        []PromptInput
}

type PromptInput struct {
	Message   string
	SortOrder int
	Weight    int
}

type ResponseGroupInput struct {
	ResponseGroupID *int64
	Name            string
	Description     *string
	SortOrder       int
	Items           []ResponseItemInput
}

type ResponseItemInput struct {
	Type      domain.ResponseItemType
	Text      *string
	URL       *string
	Caption   *string
	Metadata  json.RawMessage
	SortOrder int
}

func (s *Service) Create(ctx context.Context, staffToken string, input CreateInput) (domain.MenuOption, []domain.MenuOptionPrompt, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOption{}, nil, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.MenuOption{}, nil, ErrInvalidInput
	}
	prompts := make([]domain.MenuOptionPrompt, 0, len(input.Prompts))
	for _, prompt := range input.Prompts {
		if strings.TrimSpace(prompt.Message) == "" || prompt.Weight <= 0 {
			return domain.MenuOption{}, nil, ErrInvalidInput
		}
		prompts = append(prompts, domain.MenuOptionPrompt{Message: strings.TrimSpace(prompt.Message), SortOrder: prompt.SortOrder, Weight: prompt.Weight})
	}
	return s.store.CreateMenuOption(ctx, domain.MenuOption{ParentOptionID: input.ParentOptionID, Title: strings.TrimSpace(input.Title), Description: input.Description, SortOrder: input.SortOrder}, prompts)
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

func (s *Service) CreatePrompt(ctx context.Context, staffToken string, menuOptionID int64, input PromptInput) (domain.MenuOptionPrompt, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.MenuOptionPrompt{}, err
	}
	if menuOptionID < 1 || strings.TrimSpace(input.Message) == "" || input.Weight <= 0 {
		return domain.MenuOptionPrompt{}, ErrInvalidInput
	}
	return s.store.CreatePrompt(ctx, domain.MenuOptionPrompt{MenuOptionID: menuOptionID, Message: strings.TrimSpace(input.Message), SortOrder: input.SortOrder, Weight: input.Weight})
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

func (s *Service) ReplaceResponseGroup(ctx context.Context, staffToken string, menuOptionID int64, input ResponseGroupInput) (domain.ResponseGroup, []domain.ResponseItem, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	if menuOptionID < 1 || strings.TrimSpace(input.Name) == "" || (input.ResponseGroupID != nil && *input.ResponseGroupID < 1) {
		return domain.ResponseGroup{}, nil, ErrInvalidInput
	}
	items := make([]domain.ResponseItem, 0, len(input.Items))
	for _, item := range input.Items {
		if !validResponseItem(item) {
			return domain.ResponseGroup{}, nil, ErrInvalidInput
		}
		items = append(items, domain.ResponseItem{Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	group, createdItems, err := s.store.ReplaceResponseGroup(ctx, menuOptionID, input.ResponseGroupID, domain.ResponseGroup{Name: strings.TrimSpace(input.Name), Description: input.Description, SortOrder: input.SortOrder}, items)
	if errors.Is(err, ErrResponseGroupDoesNotBelongToOption) {
		return domain.ResponseGroup{}, nil, err
	}
	return group, createdItems, err
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

func (s *Service) GetActiveResponses(ctx context.Context, menuOptionID int64) ([]ResponseGroupDetail, error) {
	if _, err := s.store.GetActiveOption(ctx, menuOptionID); err != nil {
		return nil, ErrNotFound
	}
	groups, err := s.store.ListActiveResponseGroups(ctx, menuOptionID)
	if err != nil {
		return nil, err
	}

	result := make([]ResponseGroupDetail, 0, len(groups))
	for _, group := range groups {
		items, err := s.store.ListActiveResponseItems(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, ResponseGroupDetail{ResponseGroup: group, ResponseItems: items})
	}
	return result, nil
}
