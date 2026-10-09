package menu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"calbot/internal/domain"
	"calbot/internal/security"
)

var (
	ErrNotFound              = errors.New("menu option not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrInvalidInput          = errors.New("invalid menu option input")
	ErrMenuOptionHasResponse = errors.New("menu option already has response groups")
	ErrMenuOptionHasChildren = errors.New("menu option already has child options")
	ErrMenuOptionHasPrompts  = errors.New("menu option already has prompts")
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

type ParentOptionDetail struct {
	MenuOption  domain.MenuOption
	HasResponse bool
	HasChildren bool
	HasPrompt   bool
	IsVisible   bool
}

type SubOptionsDetail struct {
	Prompt  *domain.MenuOptionPrompt
	Options []ParentOptionDetail
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
		hasResponse, err := s.store.HasResponseGroups(ctx, *input.ParentOptionID)
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
	hasResponse, err := s.store.HasResponseGroups(ctx, menuOptionID)
	if err != nil {
		return domain.MenuOptionPrompt{}, err
	}
	if hasResponse {
		return domain.MenuOptionPrompt{}, ErrMenuOptionHasResponse
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
	hasPrompts, err := s.store.HasPrompts(ctx, menuOptionID, false)
	if err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	if hasPrompts {
		return domain.ResponseGroup{}, nil, ErrMenuOptionHasPrompts
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

func (s *Service) ListRootOptions(ctx context.Context) ([]ParentOptionDetail, error) {
	options, err := s.store.ListAllMenuOptions(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := s.evaluateMenuTree(ctx, options, nil)
	if err != nil {
		return nil, err
	}
	return publicDetailsFromNodes(nodes), nil
}

func (s *Service) GetSubOptions(ctx context.Context, id int64) (SubOptionsDetail, error) {
	option, err := s.store.GetActiveOption(ctx, id)
	if err != nil {
		return SubOptionsDetail{}, ErrNotFound
	}
	parentOptions, err := s.withPublicAvailability(ctx, []domain.MenuOption{option})
	if err != nil || len(parentOptions) == 0 || !parentOptions[0].HasPrompt {
		if err != nil {
			return SubOptionsDetail{}, err
		}
		return SubOptionsDetail{}, ErrNotFound
	}
	prompt, err := s.store.GetRandomActivePrompt(ctx, id)
	if err != nil {
		return SubOptionsDetail{}, err
	}
	children, err := s.store.ListActiveChildren(ctx, id)
	if err != nil {
		return SubOptionsDetail{}, err
	}
	options, err := s.withPublicAvailability(ctx, children)
	if err != nil {
		return SubOptionsDetail{}, err
	}
	if prompt == nil || len(options) == 0 {
		return SubOptionsDetail{}, ErrNotFound
	}
	return SubOptionsDetail{Prompt: prompt, Options: options}, nil
}

func (s *Service) withPublicAvailability(ctx context.Context, options []domain.MenuOption) ([]ParentOptionDetail, error) {
	allOptions, err := s.store.ListAllMenuOptions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ParentOptionDetail, 0, len(options))
	for _, option := range options {
		nodes, err := s.evaluateMenuTree(ctx, allOptions, option.ParentOptionID)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			if node.MenuOption.ID == option.ID && node.MenuOption.IsActive && node.IsVisible {
				result = append(result, ParentOptionDetail{MenuOption: node.MenuOption, HasResponse: node.HasResponse, HasChildren: node.HasChildren, HasPrompt: node.HasPrompts, IsVisible: node.IsVisible})
				break
			}
		}
	}
	return result, nil
}

func publicDetailsFromNodes(nodes []MenuOptionTreeNode) []ParentOptionDetail {
	result := make([]ParentOptionDetail, 0, len(nodes))
	for _, node := range nodes {
		if !node.MenuOption.IsActive || !node.IsVisible {
			continue
		}
		result = append(result, ParentOptionDetail{MenuOption: node.MenuOption, HasResponse: node.HasResponse, HasChildren: node.HasChildren, HasPrompt: node.HasPrompts, IsVisible: node.IsVisible})
	}
	return result
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
		option.HasPrompts = len(prompts) > 0
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
	hasPrompts, err := s.store.HasPrompts(ctx, id, false)
	if err != nil {
		return domain.MenuOption{}, err
	}
	option.HasPrompts = hasPrompts
	return option, nil
}

func (s *Service) ListByParentForStaff(ctx context.Context, staffToken string, parentOptionID *int64) ([]MenuOptionTreeNode, error) {
	if err := s.requireStaff(staffToken); err != nil {
		return nil, err
	}
	if parentOptionID != nil && *parentOptionID < 1 {
		return nil, ErrInvalidInput
	}
	options, err := s.store.ListAllMenuOptions(ctx)
	if err != nil {
		return nil, err
	}
	return s.evaluateMenuTree(ctx, options, parentOptionID)
}

func (s *Service) evaluateMenuTree(ctx context.Context, options []domain.MenuOption, parentOptionID *int64) ([]MenuOptionTreeNode, error) {
	childrenByParent := make(map[int64][]domain.MenuOption)
	rootOptions := make([]domain.MenuOption, 0)
	for _, option := range options {
		if option.ParentOptionID == nil {
			rootOptions = append(rootOptions, option)
			continue
		}
		childrenByParent[*option.ParentOptionID] = append(childrenByParent[*option.ParentOptionID], option)
	}

	type evaluation struct {
		hasResponse bool
		hasChildren bool
		hasPrompts  bool
		isVisible   bool
		reason      string
	}
	cache := make(map[int64]evaluation, len(options))
	visiting := make(map[int64]bool, len(options))
	var evaluate func(domain.MenuOption) (evaluation, error)
	evaluate = func(option domain.MenuOption) (evaluation, error) {
		if result, ok := cache[option.ID]; ok {
			return result, nil
		}
		if visiting[option.ID] {
			return evaluation{reason: "Se detectó una referencia circular en la rama del menú."}, nil
		}
		visiting[option.ID] = true
		defer delete(visiting, option.ID)

		children := childrenByParent[option.ID]
		result := evaluation{hasChildren: len(children) > 0}
		prompts, err := s.store.ListAllPrompts(ctx, option.ID)
		if err != nil {
			return evaluation{}, err
		}
		for _, prompt := range prompts {
			if prompt.IsActive {
				result.hasPrompts = true
				break
			}
		}

		groups, err := s.store.ListAllResponseGroups(ctx, option.ID)
		if err != nil {
			return evaluation{}, err
		}
		for _, group := range groups {
			if !group.IsActive {
				continue
			}
			items, err := s.store.ListAllResponseItems(ctx, group.ID)
			if err != nil {
				return evaluation{}, err
			}
			for _, item := range items {
				if item.IsActive {
					result.hasResponse = true
					break
				}
			}
			if result.hasResponse {
				break
			}
		}

		if !option.IsActive {
			result.reason = "La opción está inactiva."
			cache[option.ID] = result
			return result, nil
		}
		if result.hasResponse {
			result.isVisible = true
			result.reason = "La opción tiene una respuesta final activa con contenido."
			cache[option.ID] = result
			return result, nil
		}
		if !result.hasPrompts {
			switch {
			case result.hasChildren:
				result.reason = "La opción tiene subopciones, pero no tiene un prompt activo para continuar."
			case len(groups) > 0:
				result.reason = "La opción tiene grupos de respuesta, pero ninguno está activo con contenido disponible."
			default:
				result.reason = "La opción no tiene una respuesta final activa ni subopciones para continuar."
			}
			cache[option.ID] = result
			return result, nil
		}

		visibleChildren := 0
		childReasons := make([]string, 0)
		for _, child := range children {
			childResult, err := evaluate(child)
			if err != nil {
				return evaluation{}, err
			}
			if childResult.isVisible {
				visibleChildren++
				continue
			}
			childReasons = append(childReasons, fmt.Sprintf("%s: %s", child.Title, childResult.reason))
		}
		if visibleChildren > 0 {
			result.isVisible = true
			result.reason = fmt.Sprintf("La opción tiene un prompt activo y %d subopción(es) visible(s) que conducen a una respuesta final.", visibleChildren)
		} else if len(children) == 0 {
			result.reason = "La opción tiene un prompt activo, pero no tiene subopciones para continuar."
		} else {
			result.reason = "La opción tiene un prompt activo, pero ninguna subopción llega a una respuesta final. Motivos: " + strings.Join(childReasons, " | ")
		}
		cache[option.ID] = result
		return result, nil
	}

	selected := rootOptions
	if parentOptionID != nil {
		selected = childrenByParent[*parentOptionID]
	}
	result := make([]MenuOptionTreeNode, 0, len(selected))
	for _, option := range selected {
		assessment, err := evaluate(option)
		if err != nil {
			return nil, err
		}
		option.HasPrompts = assessment.hasPrompts
		result = append(result, MenuOptionTreeNode{MenuOption: option, HasResponse: assessment.hasResponse, HasChildren: assessment.hasChildren, HasPrompts: assessment.hasPrompts, IsVisible: assessment.isVisible, Reason: assessment.reason})
	}
	return result, nil
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
	availableOptions, err := s.withPublicAvailability(ctx, []domain.MenuOption{option})
	if err != nil || len(availableOptions) == 0 || !availableOptions[0].HasPrompt {
		if err != nil {
			return OptionDetail{}, err
		}
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
	availableChildren, err := s.withPublicAvailability(ctx, children)
	if err != nil {
		return OptionDetail{}, err
	}
	children = make([]domain.MenuOption, 0, len(availableChildren))
	for _, child := range availableChildren {
		children = append(children, child.MenuOption)
	}
	if prompt == nil || len(children) == 0 {
		return OptionDetail{}, ErrNotFound
	}
	option.HasPrompts = prompt != nil
	for index := range children {
		hasPrompts, err := s.store.HasPrompts(ctx, children[index].ID, true)
		if err != nil {
			return OptionDetail{}, err
		}
		children[index].HasPrompts = hasPrompts
	}
	return OptionDetail{MenuOption: option, Prompt: prompt, Children: children}, nil
}

func (s *Service) GetActiveResponse(ctx context.Context, menuOptionID int64) (*ResponseGroupDetail, error) {
	option, err := s.store.GetActiveOption(ctx, menuOptionID)
	if err != nil {
		return nil, ErrNotFound
	}
	availableOptions, err := s.withPublicAvailability(ctx, []domain.MenuOption{option})
	if err != nil {
		return nil, err
	}
	if len(availableOptions) == 0 || !availableOptions[0].HasResponse {
		return nil, ErrNotFound
	}
	group, err := s.store.GetWeightedRandomActiveResponseGroup(ctx, menuOptionID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrNotFound
	}
	items, err := s.store.ListActiveResponseItems(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &ResponseGroupDetail{ResponseGroup: *group, ResponseItems: items}, nil
}
