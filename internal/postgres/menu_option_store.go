package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"calbot/internal/domain"
	"calbot/internal/menu"
)

type MenuOptionStore struct {
	pool *pgxpool.Pool
}

func NewMenuOptionStore(pool *pgxpool.Pool) *MenuOptionStore {
	return &MenuOptionStore{pool: pool}
}

func (s *MenuOptionStore) ListActiveRootOptions(ctx context.Context) ([]domain.MenuOption, error) {
	const query = `SELECT id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at
FROM menu_options
WHERE parent_option_id IS NULL AND is_active = TRUE
ORDER BY sort_order, id`
	return scanMenuOptions(s.pool.Query(ctx, query))
}

func (s *MenuOptionStore) GetActiveOption(ctx context.Context, id int64) (domain.MenuOption, error) {
	const query = `SELECT id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at
FROM menu_options
WHERE id = $1 AND is_active = TRUE`
	return scanMenuOption(s.pool.QueryRow(ctx, query, id))
}

func (s *MenuOptionStore) GetRandomActivePrompt(ctx context.Context, menuOptionID int64) (*domain.MenuOptionPrompt, error) {
	const query = `SELECT id, menu_option_id, message, weight, is_active, created_at, updated_at
FROM menu_option_prompts
WHERE menu_option_id = $1 AND is_active = TRUE
ORDER BY -LN(1.0 - RANDOM()) / weight
LIMIT 1`
	prompt, err := scanMenuOptionPrompt(s.pool.QueryRow(ctx, query, menuOptionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

func (s *MenuOptionStore) ListActiveChildren(ctx context.Context, parentOptionID int64) ([]domain.MenuOption, error) {
	const query = `SELECT id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at
FROM menu_options
WHERE parent_option_id = $1 AND is_active = TRUE
ORDER BY sort_order, id`
	return scanMenuOptions(s.pool.Query(ctx, query, parentOptionID))
}

func (s *MenuOptionStore) GetWeightedRandomActiveResponseGroup(ctx context.Context, menuOptionID int64) (*domain.ResponseGroup, error) {
	const query = `SELECT id, menu_option_id, name, description, weight, is_active, created_at, updated_at
FROM response_groups
WHERE menu_option_id = $1 AND is_active = TRUE
ORDER BY -LN(1.0 - RANDOM()) / weight
LIMIT 1`
	var group domain.ResponseGroup
	err := s.pool.QueryRow(ctx, query, menuOptionID).Scan(&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.Weight, &group.IsActive, &group.CreatedAt, &group.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *MenuOptionStore) ListActiveResponseItems(ctx context.Context, responseGroupID int64) ([]domain.ResponseItem, error) {
	const query = `SELECT id, response_group_id, type, text, url, caption, metadata, sort_order, is_active, created_at, updated_at
FROM response_items
WHERE response_group_id = $1 AND is_active = TRUE
ORDER BY sort_order, id`
	rows, err := s.pool.Query(ctx, query, responseGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.ResponseItem, 0)
	for rows.Next() {
		var item domain.ResponseItem
		if err := rows.Scan(&item.ID, &item.ResponseGroupID, &item.Type, &item.Text, &item.URL, &item.Caption, &item.Metadata, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MenuOptionStore) CreateMenuOption(ctx context.Context, option domain.MenuOption) (domain.MenuOption, error) {
	const createOption = `INSERT INTO menu_options (parent_option_id, title, description, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at`
	if err := s.pool.QueryRow(ctx, createOption, option.ParentOptionID, option.Title, nil, option.SortOrder).Scan(
		&option.ID, &option.ParentOptionID, &option.Title, &option.Description, &option.SortOrder, &option.IsActive, &option.CreatedAt, &option.UpdatedAt,
	); err != nil {
		return domain.MenuOption{}, err
	}
	return option, nil
}

func (s *MenuOptionStore) DeactivateMenuOption(ctx context.Context, id int64) error {
	const query = `UPDATE menu_options SET is_active = FALSE WHERE id = $1 AND is_active = TRUE RETURNING id`
	var updatedID int64
	return s.pool.QueryRow(ctx, query, id).Scan(&updatedID)
}

func (s *MenuOptionStore) CreatePrompt(ctx context.Context, prompt domain.MenuOptionPrompt) (domain.MenuOptionPrompt, error) {
	const query = `INSERT INTO menu_option_prompts (menu_option_id, message, weight)
VALUES ($1, $2, $3)
RETURNING id, menu_option_id, message, weight, is_active, created_at, updated_at`
	err := s.pool.QueryRow(ctx, query, prompt.MenuOptionID, prompt.Message, prompt.Weight).Scan(
		&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	return prompt, err
}

func (s *MenuOptionStore) DeletePrompt(ctx context.Context, id int64) error {
	const query = `DELETE FROM menu_option_prompts WHERE id = $1 RETURNING id`
	var deletedID int64
	return s.pool.QueryRow(ctx, query, id).Scan(&deletedID)
}

func (s *MenuOptionStore) SetPromptActive(ctx context.Context, id int64, isActive bool) (domain.MenuOptionPrompt, error) {
	const query = `UPDATE menu_option_prompts SET is_active = $2 WHERE id = $1
RETURNING id, menu_option_id, message, weight, is_active, created_at, updated_at`
	var prompt domain.MenuOptionPrompt
	err := s.pool.QueryRow(ctx, query, id, isActive).Scan(
		&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	return prompt, err
}

func (s *MenuOptionStore) CreateResponseGroup(ctx context.Context, menuOptionID int64, group domain.ResponseGroup, items []domain.ResponseItem) (domain.ResponseGroup, []domain.ResponseItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	defer tx.Rollback(ctx)

	const lockMenuOption = `SELECT id FROM menu_options WHERE id = $1 FOR UPDATE`
	var lockedMenuOptionID int64
	if err := tx.QueryRow(ctx, lockMenuOption, menuOptionID).Scan(&lockedMenuOptionID); err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	const createGroup = `INSERT INTO response_groups (menu_option_id, name, description, weight)
VALUES ($1, $2, $3, $4)
RETURNING id, menu_option_id, name, description, weight, is_active, created_at, updated_at`
	if err := tx.QueryRow(ctx, createGroup, menuOptionID, group.Name, group.Description, group.Weight).Scan(
		&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.Weight, &group.IsActive, &group.CreatedAt, &group.UpdatedAt,
	); err != nil {
		return domain.ResponseGroup{}, nil, err
	}

	const createItem = `INSERT INTO response_items (response_group_id, type, text, url, caption, metadata, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, response_group_id, type, text, url, caption, metadata, sort_order, is_active, created_at, updated_at`
	for index := range items {
		if err := tx.QueryRow(ctx, createItem, group.ID, items[index].Type, items[index].Text, items[index].URL, items[index].Caption, items[index].Metadata, items[index].SortOrder).Scan(
			&items[index].ID, &items[index].ResponseGroupID, &items[index].Type, &items[index].Text, &items[index].URL, &items[index].Caption, &items[index].Metadata, &items[index].SortOrder, &items[index].IsActive, &items[index].CreatedAt, &items[index].UpdatedAt,
		); err != nil {
			return domain.ResponseGroup{}, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	return group, items, nil
}

func (s *MenuOptionStore) ListAllMenuOptions(ctx context.Context) ([]domain.MenuOption, error) {
	const query = `SELECT id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at
FROM menu_options
ORDER BY sort_order, id`
	return scanMenuOptions(s.pool.Query(ctx, query))
}

func (s *MenuOptionStore) GetMenuOptionByID(ctx context.Context, id int64) (domain.MenuOption, error) {
	const query = `SELECT id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at
FROM menu_options
WHERE id = $1`
	return scanMenuOption(s.pool.QueryRow(ctx, query, id))
}

func (s *MenuOptionStore) ListMenuOptionTreeNodesByParent(ctx context.Context, parentOptionID *int64) ([]menu.MenuOptionTreeNode, error) {
	if parentOptionID == nil {
		const rootQuery = `SELECT mo.id, mo.parent_option_id, mo.title, mo.description, mo.sort_order, mo.is_active, mo.created_at, mo.updated_at,
EXISTS (
    SELECT 1 FROM response_groups rg
    JOIN response_items ri ON ri.response_group_id = rg.id
    WHERE rg.menu_option_id = mo.id
) AS has_response,
EXISTS (SELECT 1 FROM menu_options child WHERE child.parent_option_id = mo.id) AS has_children
FROM menu_options mo
WHERE mo.parent_option_id IS NULL
ORDER BY mo.sort_order, mo.id`
		return scanMenuOptionTreeNodes(s.pool.Query(ctx, rootQuery))
	}

	const childQuery = `SELECT mo.id, mo.parent_option_id, mo.title, mo.description, mo.sort_order, mo.is_active, mo.created_at, mo.updated_at,
EXISTS (
    SELECT 1 FROM response_groups rg
    JOIN response_items ri ON ri.response_group_id = rg.id
    WHERE rg.menu_option_id = mo.id
 ) AS has_response,
EXISTS (SELECT 1 FROM menu_options child WHERE child.parent_option_id = mo.id) AS has_children
FROM menu_options mo
WHERE mo.parent_option_id = $1
ORDER BY mo.sort_order, mo.id`
	return scanMenuOptionTreeNodes(s.pool.Query(ctx, childQuery, *parentOptionID))
}

func scanMenuOptionTreeNodes(rows pgx.Rows, err error) ([]menu.MenuOptionTreeNode, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]menu.MenuOptionTreeNode, 0)
	for rows.Next() {
		var node menu.MenuOptionTreeNode
		if err := rows.Scan(&node.MenuOption.ID, &node.MenuOption.ParentOptionID, &node.MenuOption.Title, &node.MenuOption.Description, &node.MenuOption.SortOrder, &node.MenuOption.IsActive, &node.MenuOption.CreatedAt, &node.MenuOption.UpdatedAt, &node.HasResponse, &node.HasChildren); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (s *MenuOptionStore) DeleteMenuOption(ctx context.Context, id int64) error {
	const query = `DELETE FROM menu_options WHERE id = $1 RETURNING id`
	var deletedID int64
	return s.pool.QueryRow(ctx, query, id).Scan(&deletedID)
}

func (s *MenuOptionStore) HasResponseItems(ctx context.Context, menuOptionID int64) (bool, error) {
	const query = `SELECT EXISTS (
    SELECT 1 FROM response_groups rg
    JOIN response_items ri ON ri.response_group_id = rg.id
    WHERE rg.menu_option_id = $1
)`
	var hasResponse bool
	err := s.pool.QueryRow(ctx, query, menuOptionID).Scan(&hasResponse)
	return hasResponse, err
}

func (s *MenuOptionStore) HasChildren(ctx context.Context, menuOptionID int64) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM menu_options WHERE parent_option_id = $1)`
	var hasChildren bool
	err := s.pool.QueryRow(ctx, query, menuOptionID).Scan(&hasChildren)
	return hasChildren, err
}

func (s *MenuOptionStore) DeleteResponseGroup(ctx context.Context, id int64) error {
	const query = `DELETE FROM response_groups WHERE id = $1 RETURNING id`
	var deletedID int64
	return s.pool.QueryRow(ctx, query, id).Scan(&deletedID)
}

func (s *MenuOptionStore) ListAllPrompts(ctx context.Context, menuOptionID int64) ([]domain.MenuOptionPrompt, error) {
	const query = `SELECT id, menu_option_id, message, weight, is_active, created_at, updated_at
FROM menu_option_prompts
WHERE menu_option_id = $1
ORDER BY id`
	rows, err := s.pool.Query(ctx, query, menuOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prompts := make([]domain.MenuOptionPrompt, 0)
	for rows.Next() {
		var prompt domain.MenuOptionPrompt
		if err := rows.Scan(&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, prompt)
	}
	return prompts, rows.Err()
}

func (s *MenuOptionStore) ListAllResponseGroups(ctx context.Context, menuOptionID int64) ([]domain.ResponseGroup, error) {
	const query = `SELECT id, menu_option_id, name, description, weight, is_active, created_at, updated_at
FROM response_groups
WHERE menu_option_id = $1
ORDER BY id`
	return scanResponseGroups(s.pool.Query(ctx, query, menuOptionID))
}

func (s *MenuOptionStore) ListAllResponseItems(ctx context.Context, responseGroupID int64) ([]domain.ResponseItem, error) {
	const query = `SELECT id, response_group_id, type, text, url, caption, metadata, sort_order, is_active, created_at, updated_at
FROM response_items
WHERE response_group_id = $1
ORDER BY sort_order, id`
	return scanResponseItems(s.pool.Query(ctx, query, responseGroupID))
}

func scanResponseGroups(rows pgx.Rows, err error) ([]domain.ResponseGroup, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]domain.ResponseGroup, 0)
	for rows.Next() {
		var group domain.ResponseGroup
		if err := rows.Scan(&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.Weight, &group.IsActive, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func scanResponseItems(rows pgx.Rows, err error) ([]domain.ResponseItem, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.ResponseItem, 0)
	for rows.Next() {
		var item domain.ResponseItem
		if err := rows.Scan(&item.ID, &item.ResponseGroupID, &item.Type, &item.Text, &item.URL, &item.Caption, &item.Metadata, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type menuOptionRowScanner interface {
	Scan(dest ...any) error
}

func scanMenuOption(row menuOptionRowScanner) (domain.MenuOption, error) {
	var option domain.MenuOption
	err := row.Scan(&option.ID, &option.ParentOptionID, &option.Title, &option.Description, &option.SortOrder, &option.IsActive, &option.CreatedAt, &option.UpdatedAt)
	return option, err
}

func scanMenuOptions(rows pgx.Rows, err error) ([]domain.MenuOption, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := make([]domain.MenuOption, 0)
	for rows.Next() {
		option, err := scanMenuOption(rows)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}

func scanMenuOptionPrompt(row menuOptionRowScanner) (domain.MenuOptionPrompt, error) {
	var prompt domain.MenuOptionPrompt
	err := row.Scan(&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt)
	return prompt, err
}
