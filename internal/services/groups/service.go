package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	psnsvc "psnadd/internal/services/psn"
	"psnadd/internal/utils"
)

const (
	SortAZ     = "az"
	SortRecent = "recent"
)

type Service struct {
	pool *pgxpool.Pool
	psn  *psnsvc.Service
}

type Group struct {
	ID        int64
	Name      string
	Slug      string
	CreatedAt time.Time
}

type Entry struct {
	ID               int64
	GroupID          int64
	DisplayName      string
	OnlineID         string
	OnlineIDNorm     string
	ProfileURL       string
	AvatarURL        string
	IsPublic         *bool
	ProfileCheckedAt *time.Time
	CreatedAt        time.Time
	RemovedAt        *time.Time
	BannedAt         *time.Time
}

type CreateGroupResult struct {
	Group      Group
	GroupURL   string
	AdminURL   string
	AdminToken string
}

type AddEntryResult struct {
	Entry      Entry
	EntryURL   string
	AdminURL   string
	AdminToken string
}

type BatchAddRow struct {
	Line        int
	DisplayName string
	OnlineID    string
	Added       bool
	Error       string
	Entry       *Entry
	AdminURL    string
	AdminToken  string
}

type BatchAddResult struct {
	Rows  []BatchAddRow
	Added int
}

func New(pool *pgxpool.Pool, psn *psnsvc.Service) *Service {
	return &Service{pool: pool, psn: psn}
}

func (s *Service) CreateGroup(ctx context.Context, baseURL, name string) (CreateGroupResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CreateGroupResult{}, utils.BadRequest("group name is required")
	}
	if len(name) > 80 {
		return CreateGroupResult{}, utils.BadRequest("group name must be 80 characters or fewer")
	}

	slug, err := utils.NewToken()
	if err != nil {
		return CreateGroupResult{}, utils.Internal("could not generate group link", err)
	}
	adminToken, err := utils.NewToken()
	if err != nil {
		return CreateGroupResult{}, utils.Internal("could not generate admin link", err)
	}

	var group Group
	err = s.pool.QueryRow(ctx, `
		INSERT INTO groups (name, slug, admin_token_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, created_at
	`, name, slug, utils.HashToken(adminToken)).Scan(
		&group.ID,
		&group.Name,
		&group.Slug,
		&group.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return CreateGroupResult{}, utils.Conflict("a group with this name already exists")
		}
		return CreateGroupResult{}, utils.Internal("could not create group", err)
	}

	groupURL := fmt.Sprintf("%s/g/%s", baseURL, group.Slug)
	return CreateGroupResult{
		Group:      group,
		GroupURL:   groupURL,
		AdminURL:   groupURL + "?admin=" + adminToken,
		AdminToken: adminToken,
	}, nil
}

func (s *Service) GetGroup(ctx context.Context, slug string) (Group, error) {
	var group Group
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, created_at
		FROM groups
		WHERE slug = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(slug)).Scan(&group.ID, &group.Name, &group.Slug, &group.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Group{}, utils.NotFound("group not found")
		}
		return Group{}, utils.Internal("could not load group", err)
	}
	return group, nil
}

func (s *Service) ListEntries(ctx context.Context, groupID int64, sortBy string) ([]Entry, error) {
	orderBy := "created_at DESC"
	if sortBy == SortAZ {
		orderBy = "LOWER(online_id) ASC"
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, group_id, COALESCE(display_name, ''), online_id, online_id_norm, profile_url, COALESCE(avatar_url, ''), is_public, profile_checked_at, created_at, removed_at, banned_at
		FROM entries
		WHERE group_id = $1 AND removed_at IS NULL AND banned_at IS NULL
		ORDER BY `+orderBy+`
	`, groupID)
	if err != nil {
		return nil, utils.Internal("could not list entries", err)
	}
	defer rows.Close()

	items := []Entry{}
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, utils.Internal("could not read entry", err)
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, utils.Internal("could not list entries", err)
	}
	return items, nil
}

func (s *Service) AddEntry(ctx context.Context, baseURL string, group Group, onlineID string) (AddEntryResult, error) {
	return s.AddEntryWithName(ctx, baseURL, group, "", onlineID)
}

func (s *Service) AddEntryWithName(ctx context.Context, baseURL string, group Group, displayName, onlineID string) (AddEntryResult, error) {
	displayName = strings.TrimSpace(displayName)
	onlineID = strings.TrimSpace(onlineID)
	if !utils.ValidOnlineID(onlineID) {
		return AddEntryResult{}, utils.BadRequest("PSN ID must be 3-16 characters, start with a letter, and use only letters, numbers, underscore, or hyphen")
	}
	if len(displayName) > 120 {
		return AddEntryResult{}, utils.BadRequest("name must be 120 characters or fewer")
	}
	norm := utils.NormalizeOnlineID(onlineID)

	var blocked bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blocked_entries WHERE group_id = $1 AND online_id_norm = $2)`, group.ID, norm).Scan(&blocked); err != nil {
		return AddEntryResult{}, utils.Internal("could not check block list", err)
	}
	if blocked {
		return AddEntryResult{}, utils.Forbidden("this PSN ID is blocked for this group")
	}

	token, err := utils.NewToken()
	if err != nil {
		return AddEntryResult{}, utils.Internal("could not generate entry admin link", err)
	}

	profileURL := utils.ProfileURL(onlineID)
	profile := s.psn.Check(ctx, profileURL)

	var entry Entry
	err = s.pool.QueryRow(ctx, `
		INSERT INTO entries (group_id, display_name, online_id, online_id_norm, entry_token_hash, profile_url, avatar_url, is_public, profile_checked_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, NULLIF($7, ''), $8, NOW())
		RETURNING id, group_id, COALESCE(display_name, ''), online_id, online_id_norm, profile_url, COALESCE(avatar_url, ''), is_public, profile_checked_at, created_at, removed_at, banned_at
	`, group.ID, displayName, onlineID, norm, utils.HashToken(token), profileURL, profile.AvatarURL, profile.Public).Scan(entryDest(&entry)...)
	if err != nil {
		if isUniqueViolation(err) {
			return AddEntryResult{}, utils.Conflict("this PSN ID is already in the group")
		}
		return AddEntryResult{}, utils.Internal("could not add entry", err)
	}

	entryURL := fmt.Sprintf("%s/g/%s/%s", baseURL, group.Slug, entry.OnlineID)
	return AddEntryResult{
		Entry:      entry,
		EntryURL:   entryURL,
		AdminURL:   entryURL + "?admin=" + token,
		AdminToken: token,
	}, nil
}

func (s *Service) AddEntriesBatch(ctx context.Context, baseURL string, group Group, rows []BatchAddRow) BatchAddResult {
	result := BatchAddResult{Rows: make([]BatchAddRow, 0, len(rows))}
	for _, row := range rows {
		added, err := s.AddEntryWithName(ctx, baseURL, group, row.DisplayName, row.OnlineID)
		if err != nil {
			row.Error = utils.AsAppError(err).Message
			result.Rows = append(result.Rows, row)
			continue
		}
		row.Added = true
		row.Entry = &added.Entry
		row.AdminURL = added.AdminURL
		row.AdminToken = added.AdminToken
		result.Added++
		result.Rows = append(result.Rows, row)
	}
	return result
}

func (s *Service) GetEntry(ctx context.Context, groupID int64, onlineID string) (Entry, error) {
	var entry Entry
	err := s.pool.QueryRow(ctx, `
		SELECT id, group_id, COALESCE(display_name, ''), online_id, online_id_norm, profile_url, COALESCE(avatar_url, ''), is_public, profile_checked_at, created_at, removed_at, banned_at
		FROM entries
		WHERE group_id = $1 AND online_id_norm = $2 AND removed_at IS NULL
	`, groupID, utils.NormalizeOnlineID(onlineID)).Scan(entryDest(&entry)...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Entry{}, utils.NotFound("entry not found")
		}
		return Entry{}, utils.Internal("could not load entry", err)
	}
	return entry, nil
}

func (s *Service) PullEntry(ctx context.Context, entry Entry) (Entry, error) {
	profile := s.psn.Check(ctx, entry.ProfileURL)
	err := s.pool.QueryRow(ctx, `
		UPDATE entries
		SET is_public = $2, avatar_url = NULLIF($3, ''), profile_checked_at = NOW()
		WHERE id = $1
		RETURNING id, group_id, COALESCE(display_name, ''), online_id, online_id_norm, profile_url, COALESCE(avatar_url, ''), is_public, profile_checked_at, created_at, removed_at, banned_at
	`, entry.ID, profile.Public, profile.AvatarURL).Scan(entryDest(&entry)...)
	if err != nil {
		return Entry{}, utils.Internal("could not update profile status", err)
	}
	return entry, nil
}

func (s *Service) RemoveEntry(ctx context.Context, entry Entry) error {
	if _, err := s.pool.Exec(ctx, `UPDATE entries SET removed_at = NOW() WHERE id = $1 AND removed_at IS NULL`, entry.ID); err != nil {
		return utils.Internal("could not remove entry", err)
	}
	return nil
}

func (s *Service) BanEntry(ctx context.Context, group Group, entry Entry) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return utils.Internal("could not begin ban", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocked_entries (group_id, online_id, online_id_norm)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, online_id_norm) DO NOTHING
	`, group.ID, entry.OnlineID, entry.OnlineIDNorm); err != nil {
		return utils.Internal("could not add block", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE entries SET banned_at = NOW(), removed_at = NOW() WHERE id = $1`, entry.ID); err != nil {
		return utils.Internal("could not ban entry", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return utils.Internal("could not commit ban", err)
	}
	return nil
}

func (s *Service) GroupAdmin(ctx context.Context, groupID int64, token string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	var ok bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM groups
			WHERE id = $1 AND admin_token_hash = $2 AND deleted_at IS NULL
		)
	`, groupID, utils.HashToken(token)).Scan(&ok); err != nil {
		return false, utils.Internal("could not verify group admin", err)
	}
	return ok, nil
}

func (s *Service) EntryAdmin(ctx context.Context, entryID int64, token string) (bool, error) {
	if strings.TrimSpace(token) == "" {
		return false, nil
	}
	var ok bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM entries
			WHERE id = $1 AND entry_token_hash = $2 AND removed_at IS NULL
		)
	`, entryID, utils.HashToken(token)).Scan(&ok); err != nil {
		return false, utils.Internal("could not verify entry admin", err)
	}
	return ok, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(rows pgx.Rows) (Entry, error) {
	var entry Entry
	err := rows.Scan(entryDest(&entry)...)
	return entry, err
}

func entryDest(entry *Entry) []any {
	return []any{
		&entry.ID,
		&entry.GroupID,
		&entry.DisplayName,
		&entry.OnlineID,
		&entry.OnlineIDNorm,
		&entry.ProfileURL,
		&entry.AvatarURL,
		&entry.IsPublic,
		&entry.ProfileCheckedAt,
		&entry.CreatedAt,
		&entry.RemovedAt,
		&entry.BannedAt,
	}
}
