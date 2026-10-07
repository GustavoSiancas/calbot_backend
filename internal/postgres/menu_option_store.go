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
	const query = `SELECT id, menu_option_id, message, sort_order, weight, is_active, created_at, updated_at
FROM menu_option_prompts
WHERE menu_option_id = $1 AND is_active = TRUE
ORDER BY RANDOM()
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

func (s *MenuOptionStore) ListActiveResponseGroups(ctx context.Context, menuOptionID int64) ([]domain.ResponseGroup, error) {
	const query = `SELECT id, menu_option_id, name, description, sort_order, is_active, created_at, updated_at
FROM response_groups
WHERE menu_option_id = $1 AND is_active = TRUE
ORDER BY sort_order, id`
	rows, err := s.pool.Query(ctx, query, menuOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]domain.ResponseGroup, 0)
	for rows.Next() {
		var group domain.ResponseGroup
		if err := rows.Scan(&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.SortOrder, &group.IsActive, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
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

func (s *MenuOptionStore) CreateMenuOption(ctx context.Context, option domain.MenuOption, prompts []domain.MenuOptionPrompt) (domain.MenuOption, []domain.MenuOptionPrompt, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.MenuOption{}, nil, err
	}
	defer tx.Rollback(ctx)

	const createOption = `INSERT INTO menu_options (parent_option_id, title, description, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, parent_option_id, title, description, sort_order, is_active, created_at, updated_at`
	if err := tx.QueryRow(ctx, createOption, option.ParentOptionID, option.Title, option.Description, option.SortOrder).Scan(
		&option.ID, &option.ParentOptionID, &option.Title, &option.Description, &option.SortOrder, &option.IsActive, &option.CreatedAt, &option.UpdatedAt,
	); err != nil {
		return domain.MenuOption{}, nil, err
	}

	const createPrompt = `INSERT INTO menu_option_prompts (menu_option_id, message, sort_order, weight)
VALUES ($1, $2, $3, $4)
RETURNING id, menu_option_id, message, sort_order, weight, is_active, created_at, updated_at`
	for index := range prompts {
		if err := tx.QueryRow(ctx, createPrompt, option.ID, prompts[index].Message, prompts[index].SortOrder, prompts[index].Weight).Scan(
			&prompts[index].ID, &prompts[index].MenuOptionID, &prompts[index].Message, &prompts[index].SortOrder, &prompts[index].Weight, &prompts[index].IsActive, &prompts[index].CreatedAt, &prompts[index].UpdatedAt,
		); err != nil {
			return domain.MenuOption{}, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.MenuOption{}, nil, err
	}
	return option, prompts, nil
}

func (s *MenuOptionStore) DeactivateMenuOption(ctx context.Context, id int64) error {
	const query = `UPDATE menu_options SET is_active = FALSE WHERE id = $1 AND is_active = TRUE RETURNING id`
	var updatedID int64
	return s.pool.QueryRow(ctx, query, id).Scan(&updatedID)
}

func (s *MenuOptionStore) CreatePrompt(ctx context.Context, prompt domain.MenuOptionPrompt) (domain.MenuOptionPrompt, error) {
	const query = `INSERT INTO menu_option_prompts (menu_option_id, message, sort_order, weight)
VALUES ($1, $2, $3, $4)
RETURNING id, menu_option_id, message, sort_order, weight, is_active, created_at, updated_at`
	err := s.pool.QueryRow(ctx, query, prompt.MenuOptionID, prompt.Message, prompt.SortOrder, prompt.Weight).Scan(
		&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.SortOrder, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt,
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
RETURNING id, menu_option_id, message, sort_order, weight, is_active, created_at, updated_at`
	var prompt domain.MenuOptionPrompt
	err := s.pool.QueryRow(ctx, query, id, isActive).Scan(
		&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.SortOrder, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt,
	)
	return prompt, err
}

func (s *MenuOptionStore) ReplaceResponseGroup(ctx context.Context, menuOptionID int64, responseGroupID *int64, group domain.ResponseGroup, items []domain.ResponseItem) (domain.ResponseGroup, []domain.ResponseItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ResponseGroup{}, nil, err
	}
	defer tx.Rollback(ctx)

	if responseGroupID != nil {
		const findGroup = `SELECT menu_option_id FROM response_groups WHERE id = $1 FOR UPDATE`
		var existingMenuOptionID int64
		err := tx.QueryRow(ctx, findGroup, *responseGroupID).Scan(&existingMenuOptionID)
		if err == nil {
			if existingMenuOptionID != menuOptionID {
				return domain.ResponseGroup{}, nil, menu.ErrResponseGroupDoesNotBelongToOption
			}
			const updateGroup = `UPDATE response_groups
SET name = $2, description = $3, sort_order = $4, is_active = TRUE
WHERE id = $1
RETURNING id, menu_option_id, name, description, sort_order, is_active, created_at, updated_at`
			if err := tx.QueryRow(ctx, updateGroup, *responseGroupID, group.Name, group.Description, group.SortOrder).Scan(
				&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.SortOrder, &group.IsActive, &group.CreatedAt, &group.UpdatedAt,
			); err != nil {
				return domain.ResponseGroup{}, nil, err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM response_items WHERE response_group_id = $1`, group.ID); err != nil {
				return domain.ResponseGroup{}, nil, err
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return domain.ResponseGroup{}, nil, err
		}
	}

	if group.ID == 0 {
		const createGroup = `INSERT INTO response_groups (menu_option_id, name, description, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING id, menu_option_id, name, description, sort_order, is_active, created_at, updated_at`
		if err := tx.QueryRow(ctx, createGroup, menuOptionID, group.Name, group.Description, group.SortOrder).Scan(
			&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.SortOrder, &group.IsActive, &group.CreatedAt, &group.UpdatedAt,
		); err != nil {
			return domain.ResponseGroup{}, nil, err
		}
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

func (s *MenuOptionStore) ListAllPrompts(ctx context.Context, menuOptionID int64) ([]domain.MenuOptionPrompt, error) {
	const query = `SELECT id, menu_option_id, message, sort_order, weight, is_active, created_at, updated_at
FROM menu_option_prompts
WHERE menu_option_id = $1
ORDER BY sort_order, id`
	rows, err := s.pool.Query(ctx, query, menuOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prompts := make([]domain.MenuOptionPrompt, 0)
	for rows.Next() {
		var prompt domain.MenuOptionPrompt
		if err := rows.Scan(&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.SortOrder, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, prompt)
	}
	return prompts, rows.Err()
}

func (s *MenuOptionStore) ListAllResponseGroups(ctx context.Context, menuOptionID int64) ([]domain.ResponseGroup, error) {
	const query = `SELECT id, menu_option_id, name, description, sort_order, is_active, created_at, updated_at
FROM response_groups
WHERE menu_option_id = $1
ORDER BY sort_order, id`
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
		if err := rows.Scan(&group.ID, &group.MenuOptionID, &group.Name, &group.Description, &group.SortOrder, &group.IsActive, &group.CreatedAt, &group.UpdatedAt); err != nil {
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
	err := row.Scan(&prompt.ID, &prompt.MenuOptionID, &prompt.Message, &prompt.SortOrder, &prompt.Weight, &prompt.IsActive, &prompt.CreatedAt, &prompt.UpdatedAt)
	return prompt, err
}
