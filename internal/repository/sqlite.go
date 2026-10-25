package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	_ "modernc.org/sqlite"

	"taskmanager/models"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	if err := s.seedRoutes(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	parent_id TEXT,
	pipeline TEXT NOT NULL DEFAULT '',
	stage TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL DEFAULT 'generic',
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	priority INTEGER NOT NULL DEFAULT 0,
	depends_on TEXT NOT NULL DEFAULT '[]',
	scope TEXT NOT NULL DEFAULT '[]',
	model_hint TEXT NOT NULL DEFAULT '',
	result TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	budget_tokens INTEGER NOT NULL DEFAULT 0,
	spent_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	attempt_count INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	human_note TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	started_at TEXT,
	finished_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id);

CREATE TABLE IF NOT EXISTS attempts (
	id TEXT PRIMARY KEY,
	task_id TEXT NOT NULL,
	attempt_no INTEGER NOT NULL,
	role TEXT NOT NULL DEFAULT '',
	agent TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	model TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT '',
	verdict TEXT NOT NULL DEFAULT '',
	score INTEGER NOT NULL DEFAULT 0,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	cache_read_tokens INTEGER NOT NULL DEFAULT 0,
	cache_write_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	latency_ms INTEGER NOT NULL DEFAULT 0,
	prompt TEXT NOT NULL DEFAULT '',
	output TEXT NOT NULL DEFAULT '',
	issues TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_attempts_task ON attempts(task_id, attempt_no);

CREATE TABLE IF NOT EXISTS task_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL,
	type TEXT NOT NULL,
	from_status TEXT NOT NULL DEFAULT '',
	to_status TEXT NOT NULL DEFAULT '',
	message TEXT NOT NULL DEFAULT '',
	payload TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_task ON task_events(task_id, id);

CREATE TABLE IF NOT EXISTS context_entries (
	id TEXT PRIMARY KEY,
	root_id TEXT NOT NULL,
	task_id TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL DEFAULT 'note',
	title TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	tokens INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_context_root ON context_entries(root_id, created_at);

CREATE TABLE IF NOT EXISTS meta (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS routes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	role TEXT NOT NULL,
	position INTEGER NOT NULL,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	price_in REAL NOT NULL DEFAULT 0,
	price_out REAL NOT NULL DEFAULT 0,
	daily_tokens INTEGER NOT NULL DEFAULT 0,
	UNIQUE(role, position)
);`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return s.ensureColumns()
}

// ensureColumns adds columns introduced after the first release, so an
// existing database keeps working without a manual migration.
func (s *Store) ensureColumns() error {
	alters := []struct{ table, column, ddl string }{
		{"tasks", "scope", `ALTER TABLE tasks ADD COLUMN scope TEXT NOT NULL DEFAULT '[]'`},
		{"attempts", "cache_read_tokens", `ALTER TABLE attempts ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0`},
		{"attempts", "cache_write_tokens", `ALTER TABLE attempts ADD COLUMN cache_write_tokens INTEGER NOT NULL DEFAULT 0`},
		{"routes", "daily_tokens", `ALTER TABLE routes ADD COLUMN daily_tokens INTEGER NOT NULL DEFAULT 0`},
	}
	for _, alter := range alters {
		exists, err := s.hasColumn(alter.table, alter.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.Exec(alter.ddl); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) hasColumn(table, column string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

const routesSeedVersion = "free-v1"

var freeSeedRoutes = []models.Route{
	{Role: models.RolePlanner, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RolePlanner, Position: 1, Provider: "opencode", Model: "opencode/nemotron-3-ultra-free"},
	{Role: models.RoleGeneric, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleGeneric, Position: 1, Provider: "opencode", Model: "opencode/glm-5.3-flash-free"},
	{Role: models.RoleGeneric, Position: 2, Provider: "opencode", Model: "opencode/nemotron-3.5-lightning-free"},
	{Role: models.RoleAnalyze, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleAnalyze, Position: 1, Provider: "opencode", Model: "opencode/nemotron-3-ultra-free"},
	{Role: models.RoleDesign, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleDesign, Position: 1, Provider: "opencode", Model: "opencode/nemotron-3-ultra-free"},
	{Role: models.RoleImplement, Position: 0, Provider: "opencode", Model: "opencode/glm-5.3-flash-free"},
	{Role: models.RoleImplement, Position: 1, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleTest, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleTest, Position: 1, Provider: "opencode", Model: "opencode/flint-40b-free"},
	{Role: models.RoleReview, Position: 0, Provider: "opencode", Model: "opencode/nemotron-3-ultra-free"},
	{Role: models.RoleReview, Position: 1, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleDocs, Position: 0, Provider: "opencode", Model: "opencode/mimo-v2.6-flash-free"},
	{Role: models.RoleDocs, Position: 1, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleVerify, Position: 0, Provider: "opencode", Model: "opencode/glm-5.3-flash-free"},
	{Role: models.RoleVerify, Position: 1, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
	{Role: models.RoleAggregate, Position: 0, Provider: "opencode", Model: "opencode/deepseek-v4.1-flash-free"},
}

func (s *Store) seedRoutes() error {
	var version string
	err := s.db.QueryRow(`SELECT value FROM meta WHERE key = 'routes_seed'`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if version == routesSeedVersion {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM routes`); err != nil {
		return err
	}
	for _, r := range freeSeedRoutes {
		if _, err := tx.Exec(`INSERT INTO routes (role, position, provider, model, price_in, price_out)
			VALUES (?, ?, ?, ?, ?, ?)`, r.Role, r.Position, r.Provider, r.Model, r.PriceIn, r.PriceOut); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO meta (key, value) VALUES ('routes_seed', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, routesSeedVersion); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Routes(ctx context.Context) ([]models.Route, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT role, position, provider, model, price_in, price_out, daily_tokens FROM routes ORDER BY role, position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Route
	for rows.Next() {
		var r models.Route
		if err := rows.Scan(&r.Role, &r.Position, &r.Provider, &r.Model, &r.PriceIn, &r.PriceOut, &r.DailyTokens); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) RouteChain(ctx context.Context, role string) ([]models.Route, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT role, position, provider, model, price_in, price_out, daily_tokens
		FROM routes WHERE role = ? ORDER BY position`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Route
	for rows.Next() {
		var r models.Route
		if err := rows.Scan(&r.Role, &r.Position, &r.Provider, &r.Model, &r.PriceIn, &r.PriceOut, &r.DailyTokens); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SetRoute(ctx context.Context, role string, position int, provider, model string, priceIn, priceOut float64, dailyTokens int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO routes (role, position, provider, model, price_in, price_out, daily_tokens)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(role, position) DO UPDATE SET provider=excluded.provider, model=excluded.model,
		price_in=excluded.price_in, price_out=excluded.price_out, daily_tokens=excluded.daily_tokens`,
		role, position, provider, model, priceIn, priceOut, dailyTokens)
	return err
}

// SetRouteLimit upserts the per-day token budget for one chain entry. Zero
// means the entry has no custom budget.
func (s *Store) SetRouteLimit(ctx context.Context, role string, position int, dailyTokens int) error {
	res, err := s.db.ExecContext(ctx, `UPDATE routes SET daily_tokens = ? WHERE role = ? AND position = ?`,
		dailyTokens, role, position)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DailyModelUsage aggregates per-model token usage over the last 24 hours.
func (s *Store) DailyModelUsage(ctx context.Context) (models.ModelUsageResponse, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, model, COUNT(*),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_write_tokens), 0),
		COALESCE(SUM(cost_usd), 0), COALESCE(AVG(latency_ms), 0)
		FROM attempts WHERE status IN ('ok', 'error') AND created_at >= ?
		GROUP BY provider, model
		ORDER BY SUM(input_tokens + output_tokens) DESC, provider, model`, time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339Nano))
	if err != nil {
		return models.ModelUsageResponse{}, err
	}
	defer rows.Close()

	out := models.ModelUsageResponse{Items: []models.ModelUsage{}}
	for rows.Next() {
		var item models.ModelUsage
		var avg float64
		if err := rows.Scan(&item.Provider, &item.Model, &item.Calls, &item.InputTokens,
			&item.OutputTokens, &item.CacheReadTokens, &item.CacheWriteTokens, &item.CostUSD, &avg); err != nil {
			return models.ModelUsageResponse{}, err
		}
		item.Tokens = item.InputTokens + item.OutputTokens
		item.AvgLatencyMS = int64(math.Round(avg))
		out.Items = append(out.Items, item)
	}
	if err := rows.Err(); err != nil {
		return models.ModelUsageResponse{}, err
	}
	return out, nil
}

// ModelHealth returns the latest failed attempt per provider/model pair so
// the models page can surface quota refusals next to the chain entries.
func (s *Store) ModelHealth(ctx context.Context) (map[string]models.ModelHealth, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, model, status, error, created_at FROM attempts
		WHERE status = 'error' ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]models.ModelHealth{}
	for rows.Next() {
		var provider, model, status, lastError, created string
		if err := rows.Scan(&provider, &model, &status, &lastError, &created); err != nil {
			return nil, err
		}
		key := provider + ":" + model
		if _, dup := out[key]; dup {
			continue
		}
		var when time.Time
		parsed, perr := time.Parse(time.RFC3339Nano, created)
		if perr == nil {
			when = parsed
		}
		out[key] = models.ModelHealth{Provider: provider, Model: model, Status: status, LastError: lastError, LastErrorAt: when}
	}
	return out, nil
}

func (s *Store) DeleteRoute(ctx context.Context, role string, position int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM routes WHERE role = ? AND position = ?`, role, position)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	// Close the gap so the chain stays 0..n-1: a hole would leave the UI with
	// an entry that can neither move up nor be swapped. Positions are shifted
	// through a negative range first, otherwise the (role, position) unique
	// index fires mid-update.
	if _, err := tx.ExecContext(ctx, `UPDATE routes SET position = -(position + 1000000) WHERE role = ? AND position > ?`, role, position); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE routes SET position = -position - 1000001 WHERE role = ? AND position < 0`, role); err != nil {
		return err
	}
	return tx.Commit()
}

// MoveRoute swaps the positions of two entries in one role's chain. The
// intermediate negative position avoids tripping the (role, position) unique
// index while both rows are updated.
func (s *Store) MoveRoute(ctx context.Context, role string, from, to int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exists := func(position int) (bool, error) {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM routes WHERE role = ? AND position = ?`, role, position).Scan(&n); err != nil {
			return false, err
		}
		return n > 0, nil
	}
	okFrom, err := exists(from)
	if err != nil {
		return err
	}
	okTo, err := exists(to)
	if err != nil {
		return err
	}
	if !okFrom || !okTo {
		return ErrNotFound
	}
	temp := -1
	if from < 0 || to < 0 {
		temp = -2
	}
	if _, err := tx.ExecContext(ctx, `UPDATE routes SET position = ? WHERE role = ? AND position = ?`, temp, role, from); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE routes SET position = ? WHERE role = ? AND position = ?`, from, role, to); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE routes SET position = ? WHERE role = ? AND position = ?`, to, role, temp); err != nil {
		return err
	}
	return tx.Commit()
}

const taskCols = `id, parent_id, pipeline, stage, role, title, description, status, priority, depends_on,
	scope, model_hint, result, error, budget_tokens, spent_tokens, cost_usd, attempt_count, max_attempts,
	human_note, created_at, updated_at, started_at, finished_at`

func scanTask(sc interface{ Scan(...any) error }) (models.Task, error) {
	var t models.Task
	var parent sql.NullString
	var deps, scope string
	var created, updated string
	var started, finished sql.NullString
	err := sc.Scan(&t.ID, &parent, &t.Pipeline, &t.Stage, &t.Role, &t.Title, &t.Description, &t.Status,
		&t.Priority, &deps, &scope, &t.ModelHint, &t.Result, &t.Error, &t.BudgetTokens, &t.SpentTokens,
		&t.CostUSD, &t.AttemptCount, &t.MaxAttempts, &t.HumanNote, &created, &updated, &started, &finished)
	if err != nil {
		return t, err
	}
	if parent.Valid {
		t.ParentID = &parent.String
	}
	_ = json.Unmarshal([]byte(deps), &t.DependsOn)
	if t.DependsOn == nil {
		t.DependsOn = []string{}
	}
	_ = json.Unmarshal([]byte(scope), &t.Scope)
	if t.Scope == nil {
		t.Scope = []string{}
	}
	t.CreatedAt = parseTime(created)
	t.UpdatedAt = parseTime(updated)
	if started.Valid {
		v := parseTime(started.String)
		t.StartedAt = &v
	}
	if finished.Valid {
		v := parseTime(finished.String)
		t.FinishedAt = &v
	}
	return t, nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func fmtTime(t time.Time) string { return t.Format(time.RFC3339Nano) }

func (s *Store) CreateTask(ctx context.Context, t *models.Task) error {
	deps, _ := json.Marshal(t.DependsOn)
	if t.DependsOn == nil {
		deps = []byte("[]")
	}
	scope, _ := json.Marshal(t.Scope)
	if t.Scope == nil {
		scope = []byte("[]")
	}
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	if t.MaxAttempts <= 0 {
		t.MaxAttempts = 3
	}
	if t.Role == "" {
		t.Role = "generic"
	}
	var parent any
	if t.ParentID != nil {
		parent = *t.ParentID
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO tasks (`+taskCols+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, parent, t.Pipeline, t.Stage, t.Role, t.Title, t.Description, t.Status, t.Priority,
		string(deps), string(scope), t.ModelHint, t.Result, t.Error, t.BudgetTokens, t.SpentTokens, t.CostUSD,
		t.AttemptCount, t.MaxAttempts, t.HumanNote, fmtTime(t.CreatedAt), fmtTime(t.UpdatedAt), nil, nil)
	return err
}

func (s *Store) GetTask(ctx context.Context, id string) (models.Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) ListTasks(ctx context.Context, status string, limit int) ([]models.Task, error) {
	if limit <= 0 {
		limit = 500
	}
	q := `SELECT ` + taskCols + ` FROM tasks`
	args := []any{}
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY priority DESC, created_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) Children(ctx context.Context, parentID string) ([]models.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE parent_id = ? ORDER BY created_at ASC`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateTask(ctx context.Context, t *models.Task) error {
	deps, _ := json.Marshal(t.DependsOn)
	if t.DependsOn == nil {
		deps = []byte("[]")
	}
	scope, _ := json.Marshal(t.Scope)
	if t.Scope == nil {
		scope = []byte("[]")
	}
	t.UpdatedAt = time.Now().UTC()
	var started, finished any
	if t.StartedAt != nil {
		started = fmtTime(*t.StartedAt)
	}
	if t.FinishedAt != nil {
		finished = fmtTime(*t.FinishedAt)
	}
	var parent any
	if t.ParentID != nil {
		parent = *t.ParentID
	}
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET parent_id=?, pipeline=?, stage=?, role=?, title=?,
		description=?, status=?, priority=?, depends_on=?, scope=?, model_hint=?, result=?, error=?,
		budget_tokens=?, spent_tokens=?, cost_usd=?, attempt_count=?, max_attempts=?, human_note=?,
		updated_at=?, started_at=?, finished_at=? WHERE id=?`,
		parent, t.Pipeline, t.Stage, t.Role, t.Title, t.Description, t.Status, t.Priority,
		string(deps), string(scope), t.ModelHint, t.Result, t.Error, t.BudgetTokens, t.SpentTokens, t.CostUSD,
		t.AttemptCount, t.MaxAttempts, t.HumanNote, fmtTime(t.UpdatedAt), started, finished, t.ID)
	return err
}

func (s *Store) SetStatus(ctx context.Context, id, status string) error {
	now := time.Now().UTC()
	var started, finished any
	if status == models.StatusRunning {
		started = fmtTime(now)
	}
	if status == models.StatusDone || status == models.StatusFailed || status == models.StatusCancelled {
		finished = fmtTime(now)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status=?, updated_at=?,
		started_at=COALESCE(started_at, ?), finished_at=COALESCE(?, finished_at) WHERE id=?`,
		status, fmtTime(now), started, finished, id)
	return err
}

func (s *Store) ClaimTask(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE tasks SET status=?, updated_at=?, started_at=COALESCE(started_at, ?)
		WHERE id=? AND status IN (?, ?)`,
		models.StatusRunning, fmtTime(time.Now().UTC()), fmtTime(time.Now().UTC()), id, models.StatusPending, models.StatusRetrying)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *Store) ReadyTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskCols+` FROM tasks
		WHERE status IN (?, ?) ORDER BY priority DESC, created_at ASC LIMIT 50`, models.StatusPending, models.StatusRetrying)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) Recover(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET status=?, updated_at=? WHERE status IN (?, ?)`,
		models.StatusPending, fmtTime(time.Now().UTC()), models.StatusRunning, models.StatusVerifying)
	return err
}

func (s *Store) AddAttempt(ctx context.Context, a *models.Attempt) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO attempts (id, task_id, attempt_no, role, agent, provider, model,
		status, verdict, score, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd,
		latency_ms, prompt, output, issues, error, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.TaskID, a.AttemptNo, a.Role, a.Agent, a.Provider, a.Model, a.Status, a.Verdict,
		a.Score, a.InputTokens, a.OutputTokens, a.CacheReadTokens, a.CacheWriteTokens, a.CostUSD,
		a.LatencyMS, a.Prompt, a.Output, a.Issues, a.Error, fmtTime(a.CreatedAt))
	return err
}

// UpdateAttempt finishes a previously recorded attempt in place.
func (s *Store) UpdateAttempt(ctx context.Context, a *models.Attempt) error {
	_, err := s.db.ExecContext(ctx, `UPDATE attempts SET status=?, verdict=?, score=?, input_tokens=?,
		output_tokens=?, cache_read_tokens=?, cache_write_tokens=?, cost_usd=?, latency_ms=?, output=?, issues=?, error=? WHERE id=?`,
		a.Status, a.Verdict, a.Score, a.InputTokens, a.OutputTokens, a.CacheReadTokens, a.CacheWriteTokens,
		a.CostUSD, a.LatencyMS, a.Output, a.Issues, a.Error, a.ID)
	return err
}

func (s *Store) ListAttempts(ctx context.Context, taskID string) ([]models.Attempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, attempt_no, role, agent, provider, model, status,
		verdict, score, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, cost_usd,
		latency_ms, prompt, output, issues, error, created_at
		FROM attempts WHERE task_id = ? ORDER BY attempt_no ASC, created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Attempt{}
	for rows.Next() {
		var a models.Attempt
		var created string
		if err := rows.Scan(&a.ID, &a.TaskID, &a.AttemptNo, &a.Role, &a.Agent, &a.Provider, &a.Model,
			&a.Status, &a.Verdict, &a.Score, &a.InputTokens, &a.OutputTokens, &a.CacheReadTokens,
			&a.CacheWriteTokens, &a.CostUSD, &a.LatencyMS,
			&a.Prompt, &a.Output, &a.Issues, &a.Error, &created); err != nil {
			return nil, err
		}
		a.CreatedAt = parseTime(created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AddEvent(ctx context.Context, e *models.Event) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO task_events (task_id, type, from_status, to_status, message, payload, created_at)
		VALUES (?,?,?,?,?,?,?)`, e.TaskID, e.Type, e.FromStatus, e.ToStatus, e.Message, e.Payload, fmtTime(e.CreatedAt))
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

func (s *Store) ListEvents(ctx context.Context, taskID string, limit int) ([]models.Event, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, type, from_status, to_status, message, payload, created_at
		FROM task_events WHERE task_id = ? ORDER BY id ASC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Event{}
	for rows.Next() {
		var e models.Event
		var created string
		if err := rows.Scan(&e.ID, &e.TaskID, &e.Type, &e.FromStatus, &e.ToStatus, &e.Message, &e.Payload, &created); err != nil {
			return nil, err
		}
		e.CreatedAt = parseTime(created)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) Stats(ctx context.Context) (models.Stats, error) {
	var st models.Stats
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN status IN (?, ?) THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(cost_usd), 0), COALESCE(SUM(spent_tokens), 0) FROM tasks`,
		models.StatusRunning, models.StatusVerifying, models.StatusDone, models.StatusFailed, models.StatusNeedsHuman)
	err := row.Scan(&st.Total, &st.Running, &st.Done, &st.Failed, &st.NeedsHuman, &st.CostUSD, &st.Tokens)
	return st, err
}
