package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/client"
	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"

	"github.com/spf13/cobra"
)

const (
	agentTodoTitlePrefix = "[agent-todo] "
	todoMapFileName      = "todo-linear-map.json"
	activeIssueFileName  = "active-linear-issue.txt"
)

var bcIssueRE = regexp.MustCompile(`(?i)\bBC-\d+\b`)

func newTodosCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todos",
		Short: "Sync agent TodoWrite items to Linear subtasks",
		Long: `Mirror agent plan todos to Linear subtasks under a session parent issue.
Designed for PostToolUse / preToolUse hooks: reads TodoWrite hook JSON from stdin,
creates [agent-todo] subtasks, and syncs status changes to Linear workflow states.`,
		RunE: parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newTodosSyncCmd(flags))
	return cmd
}

func newTodosSyncCmd(flags *rootFlags) *cobra.Command {
	var parentFlag, sessionFlag, mapFileFlag, logFileFlag string
	var hookSafe bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync TodoWrite hook payload to Linear subtasks",
		Long: `Read TodoWrite hook JSON from stdin and mirror each todo to a Linear
subtask under the active parent issue. Resolves the parent from --parent,
~/.buildcore/active-linear-issue.txt, or per-session state directories.

Hook-safe by default: logs failures and exits 0 so agent sessions are never blocked.`,
		Example: `  # Hook invocation (Cursor / Claude Code PostToolUse)
  linear-pp-cli todos sync --agent

  # Manual dry-run
  printf '%s' '{"tool_name":"TodoWrite","session_id":"abc","tool_input":{"todos":[{"id":"1","content":"Ship it","status":"pending"}]}}' \
    | linear-pp-cli todos sync --parent BC-1019 --dry-run --agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if logFileFlag == "" {
				logFileFlag = filepath.Join(os.Getenv("HOME"), ".buildcore", "todo-linear-sync.log")
			}
			logger := newTodoSyncLogger(logFileFlag)

			raw, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				logger.logf("read stdin: %v", err)
				return todosSyncExit(cmd, hookSafe, flags, nil, logger)
			}
			if len(strings.TrimSpace(string(raw))) == 0 {
				return todosSyncExit(cmd, hookSafe, flags, map[string]any{"event": "todos_sync_skipped", "reason": "empty_stdin"}, logger)
			}

			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				logger.logf("invalid hook JSON: %v", err)
				return todosSyncExit(cmd, hookSafe, flags, nil, logger)
			}

			result, syncErr := runTodosSync(cmd, flags, payload, todosSyncOptions{
				parent:   parentFlag,
				session:  sessionFlag,
				mapFile:  mapFileFlag,
				logger:   logger,
			})
			if syncErr != nil {
				logger.logf("sync error: %v", syncErr)
				if !hookSafe {
					return syncErr
				}
			}
			return todosSyncExit(cmd, hookSafe, flags, result, logger)
		},
	}
	cmd.Flags().StringVar(&parentFlag, "parent", "", "Parent issue identifier (e.g. BC-1019); defaults to active-linear-issue.txt")
	cmd.Flags().StringVar(&sessionFlag, "session", "", "Session id override (defaults to hook payload session_id)")
	cmd.Flags().StringVar(&mapFileFlag, "map-file", "", "Todo id → Linear mapping file (defaults to session state dir)")
	cmd.Flags().StringVar(&logFileFlag, "log-file", "", "Sync log path (default: ~/.buildcore/todo-linear-sync.log)")
	cmd.Flags().BoolVar(&hookSafe, "hook-safe", true, "Always exit 0 and log failures (for agent hooks)")
	return cmd
}

type todosSyncOptions struct {
	parent  string
	session string
	mapFile string
	logger  *todoSyncLogger
}

type todoSyncLogger struct {
	path string
}

func newTodoSyncLogger(path string) *todoSyncLogger {
	return &todoSyncLogger{path: path}
}

func (l *todoSyncLogger) logf(format string, args ...any) {
	if l == nil || l.path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(l.path), 0o755)
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func todosSyncExit(cmd *cobra.Command, hookSafe bool, flags *rootFlags, result map[string]any, logger *todoSyncLogger) error {
	if flags.asJSON && result != nil {
		return flags.printJSON(cmd, result)
	}
	if hookSafe {
		return nil
	}
	return nil
}

type todoMapState struct {
	Parent string                       `json:"parent"`
	Items  map[string]todoMapItemRecord `json:"items"`
}

type todoMapItemRecord struct {
	LinearID string `json:"linear_id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
}

type agentTodo struct {
	ID      string
	Content string
	Status  string
}

func runTodosSync(cmd *cobra.Command, flags *rootFlags, payload map[string]any, opts todosSyncOptions) (map[string]any, error) {
	tool := firstString(payload, "tool_name", "toolName", "tool", "name")
	if tool != "" && !strings.EqualFold(tool, "TodoWrite") && tool != "todo_write" && tool != "todoWrite" {
		return map[string]any{"event": "todos_sync_skipped", "reason": "not_todo_write", "tool": tool}, nil
	}

	todos := extractTodos(payload)
	if len(todos) == 0 {
		return map[string]any{"event": "todos_sync_skipped", "reason": "no_todos"}, nil
	}

	session := opts.session
	if session == "" {
		session = firstString(payload, "session_id", "sessionId", "conversation_id", "conversationId")
	}
	if session == "" {
		session = "unknown-session"
	}

	parent := opts.parent
	if parent == "" {
		parent = readActiveParent(session, todos)
	}
	if parent == "" {
		opts.logger.logf("skip session %s: no active BC parent (save active-linear-issue.txt)", shortSession(session))
		return map[string]any{"event": "todos_sync_skipped", "reason": "no_parent", "session": session}, nil
	}
	parent = strings.ToUpper(parent)

	mapPath := opts.mapFile
	if mapPath == "" {
		mapPath = defaultTodoMapPath(session)
	}
	state, err := loadTodoMap(mapPath)
	if err != nil {
		return nil, err
	}
	if state.Parent == "" {
		state.Parent = parent
	} else if state.Parent != parent {
		opts.logger.logf("parent drift session %s: map=%s active=%s", shortSession(session), state.Parent, parent)
	}
	if state.Items == nil {
		state.Items = map[string]todoMapItemRecord{}
	}

	if flags.dryRun {
		preview := map[string]any{
			"event":   "would_sync_todos",
			"parent":  parent,
			"session": session,
			"todos":   len(todos),
		}
		return preview, nil
	}

	c, err := flags.newClient()
	if err != nil {
		return nil, err
	}

	parentIssue, err := fetchParentIssue(c, parent)
	if err != nil {
		return nil, err
	}

	sessTag := resolvePPSession(flags, session)
	if sessTag == "" || sessTag == "current" {
		sessTag = ppCurrentSession()
	}

	created := []string{}
	updated := []string{}
	for _, todo := range todos {
		if todo.ID == "" {
			continue
		}
		record, ok := state.Items[todo.ID]
		if !ok {
			ident, createErr := createAgentTodoSubtask(c, flags, parentIssue, todo, sessTag)
			if createErr != nil {
				opts.logger.logf("subtask failed (%s): %v", parent, createErr)
				continue
			}
			record = todoMapItemRecord{LinearID: ident, Content: todo.Content, Status: todo.Status}
			state.Items[todo.ID] = record
			created = append(created, ident)
			opts.logger.logf("created %s under %s: %s", ident, parent, truncate(todo.Content, 120))
			if todo.Status == "in_progress" || todo.Status == "completed" || todo.Status == "cancelled" {
				if syncErr := syncTodoStatus(c, ident, parentIssue.TeamID, todo.Status, opts.logger); syncErr != nil {
					opts.logger.logf("status sync failed (%s -> %s): %v", ident, todo.Status, syncErr)
				}
			}
			continue
		}

		ident := record.LinearID
		if todo.Content != "" && todo.Content != record.Content && ident != "" {
			if noteErr := addTodoUpdateComment(c, ident, todo.Content); noteErr != nil {
				opts.logger.logf("note failed (%s): %v", ident, noteErr)
			} else {
				record.Content = todo.Content
				state.Items[todo.ID] = record
			}
		}
		if todo.Status != record.Status && ident != "" {
			if syncErr := syncTodoStatus(c, ident, parentIssue.TeamID, todo.Status, opts.logger); syncErr != nil {
				opts.logger.logf("status sync failed (%s -> %s): %v", ident, todo.Status, syncErr)
			} else {
				record.Status = todo.Status
				state.Items[todo.ID] = record
				updated = append(updated, ident)
			}
		}
	}

	if err := saveTodoMap(mapPath, state); err != nil {
		opts.logger.logf("save map: %v", err)
	}

	return map[string]any{
		"event":   "todos_synced",
		"parent":  parent,
		"session": session,
		"created": created,
		"updated": updated,
	}, nil
}

type parentIssueInfo struct {
	ID     string
	TeamID string
	TeamKey string
}

func fetchParentIssue(c *client.Client, identifier string) (parentIssueInfo, error) {
	raw, err := fetchIssueLive(c, identifier)
	if err != nil {
		return parentIssueInfo{}, err
	}
	var issue struct {
		ID   string `json:"id"`
		Team struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"team"`
	}
	if err := json.Unmarshal(raw, &issue); err != nil {
		return parentIssueInfo{}, fmt.Errorf("parsing parent issue: %w", err)
	}
	if issue.ID == "" || issue.Team.ID == "" {
		return parentIssueInfo{}, fmt.Errorf("parent %q missing team", identifier)
	}
	return parentIssueInfo{ID: issue.ID, TeamID: issue.Team.ID, TeamKey: issue.Team.Key}, nil
}

func createAgentTodoSubtask(c *client.Client, flags *rootFlags, parent parentIssueInfo, todo agentTodo, session string) (string, error) {
	title := agentTodoTitle(todo.Content)
	desc := agentTodoDescription(todo, session)
	input := map[string]any{
		"teamId":      parent.TeamID,
		"parentId":    parent.ID,
		"title":       title,
		"description": desc,
		"priority":    3,
	}

	const mutation = `mutation CreateIssue($input: IssueCreateInput!) {
		issueCreate(input: $input) {
			success
			issue { id identifier title url }
		}
	}`
	resp, err := c.Mutate(mutation, map[string]any{"input": input})
	if err != nil {
		return "", classifyMutationError("issueCreate", err, flags, nil)
	}
	var parsed struct {
		IssueCreate struct {
			Success bool `json:"success"`
			Issue   struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
			} `json:"issue"`
		} `json:"issueCreate"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return "", fmt.Errorf("parsing issueCreate response: %w", err)
	}
	if !parsed.IssueCreate.Success {
		return "", apiErr(fmt.Errorf("Linear reported issueCreate success=false"))
	}

	dbPath := defaultDBPath("linear-pp-cli")
	if db, dbErr := store.Open(dbPath); dbErr == nil {
		defer db.Close()
		_ = db.RecordPPFixture(parsed.IssueCreate.Issue.ID, parsed.IssueCreate.Issue.Identifier, parsed.IssueCreate.Issue.Title, session)
	}

	return strings.ToUpper(parsed.IssueCreate.Issue.Identifier), nil
}

func syncTodoStatus(c *client.Client, ident, teamID, status string, logger *todoSyncLogger) error {
	switch status {
	case "in_progress":
		return setIssueStateByType(c, ident, teamID, "started")
	case "completed":
		return setIssueStateByType(c, ident, teamID, "completed")
	case "cancelled":
		return addTodoUpdateComment(c, ident, "Agent TodoWrite item cancelled in session.")
	default:
		return nil
	}
}

func setIssueStateByType(c *client.Client, ident, teamID, stateType string) error {
	stateID, err := resolveWorkflowStateID(c, teamID, stateType)
	if err != nil {
		return err
	}
	issueID, err := resolveIssueID(c, ident)
	if err != nil {
		return err
	}
	const mutation = `mutation($id: String!, $input: IssueUpdateInput!) {
		issueUpdate(id: $id, input: $input) { success }
	}`
	resp, err := c.Mutate(mutation, map[string]any{"id": issueID, "input": map[string]any{"stateId": stateID}})
	if err != nil {
		return err
	}
	var parsed struct {
		IssueUpdate struct {
			Success bool `json:"success"`
		} `json:"issueUpdate"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return fmt.Errorf("parsing issueUpdate response: %w", err)
	}
	if !parsed.IssueUpdate.Success {
		return apiErr(fmt.Errorf("Linear reported issueUpdate success=false"))
	}
	return nil
}

func resolveWorkflowStateID(c graphqlQueryer, teamID, stateType string) (string, error) {
	const query = `query($teamId: ID!, $type: String!) {
		workflowStates(filter: { team: { id: { eq: $teamId } }, type: { eq: $type } }, first: 10) {
			nodes { id name type position }
		}
	}`
	var resp struct {
		WorkflowStates struct {
			Nodes []struct {
				ID       string `json:"id"`
				Position float64 `json:"position"`
			} `json:"nodes"`
		} `json:"workflowStates"`
	}
	if err := c.QueryInto(query, map[string]any{"teamId": teamID, "type": stateType}, &resp); err != nil {
		return "", err
	}
	if len(resp.WorkflowStates.Nodes) == 0 {
		return "", fmt.Errorf("no workflow state of type %q for team", stateType)
	}
	best := resp.WorkflowStates.Nodes[0]
	for _, node := range resp.WorkflowStates.Nodes[1:] {
		if node.Position < best.Position {
			best = node
		}
	}
	return best.ID, nil
}

func addTodoUpdateComment(c *client.Client, ident, body string) error {
	issueID, err := resolveIssueID(c, ident)
	if err != nil {
		return err
	}
	const mutation = `mutation($input: CommentCreateInput!) {
		commentCreate(input: $input) { success }
	}`
	resp, err := c.Mutate(mutation, map[string]any{"input": map[string]any{
		"issueId": issueID,
		"body":    body,
	}})
	if err != nil {
		return err
	}
	var parsed struct {
		CommentCreate struct {
			Success bool `json:"success"`
		} `json:"commentCreate"`
	}
	if err := json.Unmarshal(resp, &parsed); err != nil {
		return fmt.Errorf("parsing commentCreate response: %w", err)
	}
	if !parsed.CommentCreate.Success {
		return apiErr(fmt.Errorf("Linear reported commentCreate success=false"))
	}
	return nil
}

func agentTodoTitle(content string) string {
	body := strings.Join(strings.Fields(content), " ")
	if body == "" {
		return agentTodoTitlePrefix + "Untitled todo"
	}
	if len(body) <= 180 {
		return agentTodoTitlePrefix + body
	}
	return agentTodoTitlePrefix + body[:177] + "..."
}

func agentTodoDescription(todo agentTodo, session string) string {
	content := strings.TrimSpace(todo.Content)
	if content == "" {
		content = "Untitled todo"
	}
	return fmt.Sprintf("## Goal\n%s\n\n## Acceptance\nAgent TodoWrite item marked completed in session.\n\n## Context links\nTodo id: %s\n\n_Created by session %s via `linear-pp-cli todos sync`._",
		content, todo.ID, session)
}

func extractTodos(payload map[string]any) []agentTodo {
	input := toolInputMap(payload)
	raw, _ := input["todos"].([]any)
	out := make([]agentTodo, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, agentTodo{
			ID:      strings.TrimSpace(fmt.Sprint(m["id"])),
			Content: strings.TrimSpace(fmt.Sprint(m["content"])),
			Status:  strings.ToLower(strings.TrimSpace(fmt.Sprint(m["status"]))),
		})
	}
	return out
}

func toolInputMap(payload map[string]any) map[string]any {
	for _, key := range []string{"tool_input", "toolInput", "input", "arguments", "tool_arguments"} {
		val, ok := payload[key]
		if !ok {
			continue
		}
		if m, ok := val.(map[string]any); ok {
			return m
		}
		if s, ok := val.(string); ok {
			var parsed map[string]any
			if json.Unmarshal([]byte(s), &parsed) == nil {
				return parsed
			}
		}
	}
	return map[string]any{}
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		val, ok := data[key]
		if !ok {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(val))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func readActiveParent(session string, todos []agentTodo) string {
	home := os.Getenv("HOME")
	globalPath := filepath.Join(home, ".buildcore", activeIssueFileName)
	if ident := readBCFromFile(globalPath); ident != "" {
		return ident
	}
	for _, dir := range sessionStateDirs(home, session) {
		if ident := readBCFromFile(filepath.Join(dir, activeIssueFileName)); ident != "" {
			return ident
		}
	}
	for _, todo := range todos {
		if m := bcIssueRE.FindString(todo.Content); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}

func readBCFromFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if m := bcIssueRE.FindString(string(raw)); m != "" {
		return strings.ToUpper(m)
	}
	return ""
}

func sessionStateDirs(home, session string) []string {
	candidates := []string{
		filepath.Join(home, ".claude", "state", session),
		filepath.Join(home, ".cursor", "state", session),
		filepath.Join(home, ".codex", "state", session),
	}
	found := []string{}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			found = append(found, dir)
		}
	}
	if len(found) > 0 {
		return found
	}
	return candidates
}

func defaultTodoMapPath(session string) string {
	dirs := sessionStateDirs(os.Getenv("HOME"), session)
	dir := dirs[0]
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, todoMapFileName)
}

func loadTodoMap(path string) (todoMapState, error) {
	state := todoMapState{Items: map[string]todoMapItemRecord{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return todoMapState{Items: map[string]todoMapItemRecord{}}, nil
	}
	if state.Items == nil {
		state.Items = map[string]todoMapItemRecord{}
	}
	return state, nil
}

func saveTodoMap(path string, state todoMapState) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func shortSession(session string) string {
	if len(session) <= 8 {
		return session
	}
	return session[:8]
}
