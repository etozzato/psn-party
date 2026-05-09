package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"psnadd/internal/config"
	"psnadd/internal/db"
	"psnadd/internal/utils"
)

type groupRow struct {
	ID        int64
	Name      string
	Slug      string
	Entries   int64
	Blocked   int64
	CreatedAt time.Time
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.OpenPool(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "database migrations failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PSN Add Admin Console")
	fmt.Println("Type 'help' for commands.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("psn-admin> ")
		if !scanner.Scan() {
			fmt.Println()
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := run(ctx, pool, line); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}
}

func run(ctx context.Context, pool *pgxpool.Pool, line string) error {
	args := strings.Fields(line)
	switch strings.ToLower(args[0]) {
	case "help", "h", "?":
		help()
	case "exit", "quit", "q":
		os.Exit(0)
	case "list", "ls":
		return listGroups(ctx, pool)
	case "show":
		if len(args) != 2 {
			return errors.New("usage: show <group-id-or-slug>")
		}
		return showGroup(ctx, pool, args[1])
	case "rename":
		if len(args) < 3 {
			return errors.New("usage: rename <group-id-or-slug> <new name>")
		}
		return renameGroup(ctx, pool, args[1], strings.Join(args[2:], " "))
	case "delete":
		if len(args) != 2 {
			return errors.New("usage: delete <group-id-or-slug>")
		}
		return deleteGroup(ctx, pool, args[1])
	case "ban":
		if len(args) != 3 {
			return errors.New("usage: ban <group-id-or-slug> <psn-id>")
		}
		return banID(ctx, pool, args[1], args[2])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
	return nil
}

func help() {
	fmt.Println("Commands:")
	fmt.Println("  list")
	fmt.Println("  show <group-id-or-slug>")
	fmt.Println("  rename <group-id-or-slug> <new name>")
	fmt.Println("  delete <group-id-or-slug>")
	fmt.Println("  ban <group-id-or-slug> <psn-id>")
	fmt.Println("  exit")
}

func listGroups(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT g.id, g.name, g.slug, g.created_at,
			COUNT(DISTINCT e.id) FILTER (WHERE e.removed_at IS NULL AND e.banned_at IS NULL) AS entries,
			COUNT(DISTINCT b.id) AS blocked
		FROM groups g
		LEFT JOIN entries e ON e.group_id = g.id
		LEFT JOIN blocked_entries b ON b.group_id = g.id
		WHERE g.deleted_at IS NULL
		GROUP BY g.id
		ORDER BY g.created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tNAME\tENTRIES\tBLOCKED\tSLUG")
	for rows.Next() {
		var row groupRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Slug, &row.CreatedAt, &row.Entries, &row.Blocked); err != nil {
			return err
		}
		fmt.Fprintf(writer, "%d\t%s\t%d\t%d\t%s\n", row.ID, row.Name, row.Entries, row.Blocked, row.Slug)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return writer.Flush()
}

func showGroup(ctx context.Context, pool *pgxpool.Pool, ref string) error {
	id, err := groupID(ctx, pool, ref)
	if err != nil {
		return err
	}
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(display_name, ''), online_id, is_public, created_at, removed_at, banned_at
		FROM entries
		WHERE group_id = $1
		ORDER BY created_at DESC
	`, id)
	if err != nil {
		return err
	}
	defer rows.Close()

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tPSN_ID\tPUBLIC\tCREATED\tREMOVED\tBANNED")
	for rows.Next() {
		var displayName string
		var onlineID string
		var isPublic *bool
		var created time.Time
		var removed, banned *time.Time
		if err := rows.Scan(&displayName, &onlineID, &isPublic, &created, &removed, &banned); err != nil {
			return err
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n", displayName, onlineID, publicText(isPublic), created.Format(time.RFC3339), timeText(removed), timeText(banned))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return writer.Flush()
}

func publicText(value *bool) string {
	if value == nil {
		return "unchecked"
	}
	if *value {
		return "public"
	}
	return "private"
}

func timeText(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.Format(time.RFC3339)
}

func renameGroup(ctx context.Context, pool *pgxpool.Pool, ref, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return errors.New("name must be 1-80 characters")
	}
	id, err := groupID(ctx, pool, ref)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `UPDATE groups SET name = $2 WHERE id = $1 AND deleted_at IS NULL`, id, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("group not found")
	}
	fmt.Println("renamed")
	return nil
}

func deleteGroup(ctx context.Context, pool *pgxpool.Pool, ref string) error {
	id, err := groupID(ctx, pool, ref)
	if err != nil {
		return err
	}
	tag, err := pool.Exec(ctx, `UPDATE groups SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("group not found")
	}
	fmt.Println("deleted")
	return nil
}

func banID(ctx context.Context, pool *pgxpool.Pool, ref, onlineID string) error {
	if !utils.ValidOnlineID(onlineID) {
		return errors.New("invalid PSN ID")
	}
	id, err := groupID(ctx, pool, ref)
	if err != nil {
		return err
	}
	norm := utils.NormalizeOnlineID(onlineID)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocked_entries (group_id, online_id, online_id_norm)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, online_id_norm) DO NOTHING
	`, id, onlineID, norm); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE entries SET removed_at = NOW(), banned_at = NOW() WHERE group_id = $1 AND online_id_norm = $2`, id, norm); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("banned")
	return nil
}

func groupID(ctx context.Context, pool *pgxpool.Pool, ref string) (int64, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return id, nil
	}
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM groups WHERE slug = $1 AND deleted_at IS NULL`, ref).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("group not found")
		}
		return 0, err
	}
	return id, nil
}
