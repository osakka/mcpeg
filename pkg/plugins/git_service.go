package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
)

// GitService provides git operations for development workflows
type GitService struct {
	*BasePlugin
	workingDir string
	gitPath    string
}

// NewGitService creates a new git service plugin
func NewGitService() *GitService {
	return &GitService{
		BasePlugin: NewBasePlugin(
			"git",
			"1.0.0",
			"Git version control operations for development workflows",
		),
		gitPath: "git", // Default git command
	}
}

// Initialize initializes the git service
func (gs *GitService) Initialize(ctx context.Context, config PluginConfig) error {
	if err := gs.BasePlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Set working directory
	gs.workingDir = "."
	if configWorkDir, ok := config.Config["working_dir"].(string); ok {
		gs.workingDir = configWorkDir
	}

	// Set git path if configured
	if configGitPath, ok := config.Config["git_path"].(string); ok {
		gs.gitPath = configGitPath
	}

	// Verify git is available
	if err := gs.verifyGitAvailable(); err != nil {
		return fmt.Errorf("git not available: %w", err)
	}

	gs.logger.Info("git_service_initialized",
		"working_dir", gs.workingDir,
		"git_path", gs.gitPath)

	return nil
}

// GetTools returns the tools provided by the git service
func (gs *GitService) GetTools() []registry.ToolDefinition {
	return []registry.ToolDefinition{
		{
			Name:        "git_status",
			Description: "Get git repository status",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"porcelain": map[string]interface{}{
						"type":        "boolean",
						"description": "Use porcelain format for machine-readable output",
						"default":     false,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Get repository status",
					Input:       map[string]interface{}{},
				},
			},
		},
		{
			Name:        "git_diff",
			Description: "Show changes between commits, commit and working tree, etc",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"staged": map[string]interface{}{
						"type":        "boolean",
						"description": "Show staged changes only",
						"default":     false,
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Show diff for specific file",
					},
					"commit": map[string]interface{}{
						"type":        "string",
						"description": "Compare with specific commit",
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Show all changes",
					Input:       map[string]interface{}{},
				},
				{
					Description: "Show staged changes only",
					Input: map[string]interface{}{
						"staged": true,
					},
				},
			},
		},
		{
			Name:        "git_add",
			Description: "Add file contents to the staging area",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Files to add (use '.' for all)",
					},
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "Add all tracked files",
						"default":     false,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Add specific files",
					Input: map[string]interface{}{
						"files": []string{"src/main.go", "README.md"},
					},
				},
				{
					Description: "Add all changes",
					Input: map[string]interface{}{
						"all": true,
					},
				},
			},
		},
		{
			Name:        "git_commit",
			Description: "Record changes to the repository",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Commit message",
					},
					"amend": map[string]interface{}{
						"type":        "boolean",
						"description": "Amend the last commit",
						"default":     false,
					},
				},
				"required": []string{"message"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Create a commit",
					Input: map[string]interface{}{
						"message": "Add new feature implementation",
					},
				},
			},
		},
		{
			Name:        "git_push",
			Description: "Update remote refs along with associated objects",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"remote": map[string]interface{}{
						"type":        "string",
						"description": "Remote name (default: origin)",
						"default":     "origin",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch name (default: current branch)",
					},
					"force": map[string]interface{}{
						"type":        "boolean",
						"description": "Force push (use with caution)",
						"default":     false,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Push to origin",
					Input:       map[string]interface{}{},
				},
			},
		},
		{
			Name:        "git_pull",
			Description: "Fetch from and integrate with another repository or a local branch",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"remote": map[string]interface{}{
						"type":        "string",
						"description": "Remote name (default: origin)",
						"default":     "origin",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch name (default: current branch)",
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Pull from origin",
					Input:       map[string]interface{}{},
				},
			},
		},
		{
			Name:        "git_branch",
			Description: "List, create, or delete branches",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"list", "create", "delete", "switch"},
						"description": "Action to perform",
						"default":     "list",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Branch name (required for create/delete/switch)",
					},
					"force": map[string]interface{}{
						"type":        "boolean",
						"description": "Force delete branch",
						"default":     false,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "List branches",
					Input: map[string]interface{}{
						"action": "list",
					},
				},
				{
					Description: "Create new branch",
					Input: map[string]interface{}{
						"action": "create",
						"name":   "feature/new-feature",
					},
				},
			},
		},
		{
			Name:        "git_log",
			Description: "Show commit logs",
			Category:    "git",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of commits to show",
						"default":     10,
						"minimum":     1,
						"maximum":     100,
					},
					"oneline": map[string]interface{}{
						"type":        "boolean",
						"description": "Show one line per commit",
						"default":     false,
					},
					"graph": map[string]interface{}{
						"type":        "boolean",
						"description": "Show commit graph",
						"default":     false,
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Show recent commits",
					Input: map[string]interface{}{
						"limit": 5,
					},
				},
			},
		},
	}
}

// GetResources returns resources provided by the git service
func (gs *GitService) GetResources() []registry.ResourceDefinition {
	return []registry.ResourceDefinition{
		{
			Name:        "git_repo_info",
			Type:        "application/json",
			Description: "Git repository information and metadata",
		},
		{
			Name:        "git_remote_info",
			Type:        "application/json",
			Description: "Information about git remotes",
		},
	}
}

// GetPrompts returns prompts provided by the git service
func (gs *GitService) GetPrompts() []registry.PromptDefinition {
	return []registry.PromptDefinition{
		{
			Name:        "git_workflow",
			Description: "Suggest git workflow based on repository state",
			Category:    "workflow",
		},
		{
			Name:        "commit_message",
			Description: "Generate commit message based on changes",
			Category:    "automation",
		},
	}
}

// CallTool executes a git service tool
func (gs *GitService) CallTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	start := time.Now()
	defer func() {
		gs.LogToolCall(name, time.Since(start), nil)
	}()

	switch name {
	case "git_status":
		return gs.handleStatus(args)
	case "git_diff":
		return gs.handleDiff(args)
	case "git_add":
		return gs.handleAdd(args)
	case "git_commit":
		return gs.handleCommit(args)
	case "git_push":
		return gs.handlePush(args)
	case "git_pull":
		return gs.handlePull(args)
	case "git_branch":
		return gs.handleBranch(args)
	case "git_log":
		return gs.handleLog(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ReadResource reads a git service resource
func (gs *GitService) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	start := time.Now()
	defer func() {
		gs.LogResourceAccess(uri, time.Since(start), nil)
	}()

	switch uri {
	case "git_repo_info":
		return gs.getRepoInfo()
	case "git_remote_info":
		return gs.getRemoteInfo()
	default:
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

// ListResources lists available resources
func (gs *GitService) ListResources(ctx context.Context) ([]registry.ResourceDefinition, error) {
	return gs.GetResources(), nil
}

// GetPrompt returns a prompt
func (gs *GitService) GetPrompt(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	switch name {
	case "git_workflow":
		return gs.handleWorkflowPrompt(args)
	case "commit_message":
		return gs.handleCommitMessagePrompt(args)
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

// Tool handlers

func (gs *GitService) handleStatus(args json.RawMessage) (interface{}, error) {
	var req struct {
		Porcelain bool `json:"porcelain"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	gitArgs := []string{"status"}
	if req.Porcelain {
		gitArgs = append(gitArgs, "--porcelain")
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_status_operations_total")

	return map[string]interface{}{
		"output":    output,
		"porcelain": req.Porcelain,
	}, nil
}

func (gs *GitService) handleDiff(args json.RawMessage) (interface{}, error) {
	var req struct {
		Staged bool    `json:"staged"`
		File   *string `json:"file,omitempty"`
		Commit *string `json:"commit,omitempty"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	gitArgs := []string{"diff"}

	if req.Staged {
		gitArgs = append(gitArgs, "--staged")
	}

	if req.Commit != nil {
		gitArgs = append(gitArgs, *req.Commit)
	}

	if req.File != nil {
		gitArgs = append(gitArgs, "--", *req.File)
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_diff_operations_total")

	return map[string]interface{}{
		"diff":   output,
		"staged": req.Staged,
		"file":   req.File,
	}, nil
}

func (gs *GitService) handleAdd(args json.RawMessage) (interface{}, error) {
	var req struct {
		Files []string `json:"files,omitempty"`
		All   bool     `json:"all"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	gitArgs := []string{"add"}

	if req.All {
		gitArgs = append(gitArgs, "-A")
	} else if len(req.Files) > 0 {
		gitArgs = append(gitArgs, req.Files...)
	} else {
		return nil, fmt.Errorf("must specify files or set all=true")
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_add_operations_total")

	return map[string]interface{}{
		"success":     true,
		"output":      output,
		"files_added": len(req.Files),
	}, nil
}

func (gs *GitService) handleCommit(args json.RawMessage) (interface{}, error) {
	var req struct {
		Message string `json:"message"`
		Amend   bool   `json:"amend"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if req.Message == "" {
		return nil, fmt.Errorf("commit message cannot be empty")
	}

	gitArgs := []string{"commit", "-m", req.Message}

	if req.Amend {
		gitArgs = append(gitArgs, "--amend")
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_commit_operations_total")

	return map[string]interface{}{
		"success": true,
		"output":  output,
		"message": req.Message,
		"amend":   req.Amend,
	}, nil
}

func (gs *GitService) handlePush(args json.RawMessage) (interface{}, error) {
	var req struct {
		Remote *string `json:"remote,omitempty"`
		Branch *string `json:"branch,omitempty"`
		Force  bool    `json:"force"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	gitArgs := []string{"push"}

	if req.Force {
		gitArgs = append(gitArgs, "--force")
	}

	remote := "origin"
	if req.Remote != nil {
		remote = *req.Remote
	}
	gitArgs = append(gitArgs, remote)

	if req.Branch != nil {
		gitArgs = append(gitArgs, *req.Branch)
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_push_operations_total")

	return map[string]interface{}{
		"success": true,
		"output":  output,
		"remote":  remote,
		"force":   req.Force,
	}, nil
}

func (gs *GitService) handlePull(args json.RawMessage) (interface{}, error) {
	var req struct {
		Remote *string `json:"remote,omitempty"`
		Branch *string `json:"branch,omitempty"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	gitArgs := []string{"pull"}

	if req.Remote != nil {
		gitArgs = append(gitArgs, *req.Remote)
		if req.Branch != nil {
			gitArgs = append(gitArgs, *req.Branch)
		}
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_pull_operations_total")

	return map[string]interface{}{
		"success": true,
		"output":  output,
	}, nil
}

func (gs *GitService) handleBranch(args json.RawMessage) (interface{}, error) {
	var req struct {
		Action string  `json:"action"`
		Name   *string `json:"name,omitempty"`
		Force  bool    `json:"force"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	var gitArgs []string

	switch req.Action {
	case "list", "":
		gitArgs = []string{"branch", "-a"}
	case "create":
		if req.Name == nil {
			return nil, fmt.Errorf("branch name required for create action")
		}
		gitArgs = []string{"branch", *req.Name}
	case "delete":
		if req.Name == nil {
			return nil, fmt.Errorf("branch name required for delete action")
		}
		if req.Force {
			gitArgs = []string{"branch", "-D", *req.Name}
		} else {
			gitArgs = []string{"branch", "-d", *req.Name}
		}
	case "switch":
		if req.Name == nil {
			return nil, fmt.Errorf("branch name required for switch action")
		}
		gitArgs = []string{"checkout", *req.Name}
	default:
		return nil, fmt.Errorf("invalid action: %s", req.Action)
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_branch_operations_total", "action", req.Action)

	return map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"output":  output,
	}, nil
}

func (gs *GitService) handleLog(args json.RawMessage) (interface{}, error) {
	var req struct {
		Limit   int  `json:"limit"`
		Oneline bool `json:"oneline"`
		Graph   bool `json:"graph"`
	}

	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	gitArgs := []string{"log", fmt.Sprintf("-%d", req.Limit)}

	if req.Oneline {
		gitArgs = append(gitArgs, "--oneline")
	}

	if req.Graph {
		gitArgs = append(gitArgs, "--graph")
	}

	output, err := gs.execGit(gitArgs...)
	if err != nil {
		return nil, err
	}

	gs.metrics.Inc("git_log_operations_total")

	return map[string]interface{}{
		"log":     output,
		"limit":   req.Limit,
		"oneline": req.Oneline,
		"graph":   req.Graph,
	}, nil
}

// Helper methods

func (gs *GitService) verifyGitAvailable() error {
	_, err := gs.execGit("--version")
	return err
}

func (gs *GitService) execGit(args ...string) (string, error) {
	cmd := exec.Command(gs.gitPath, args...)
	cmd.Dir = gs.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

func (gs *GitService) getRepoInfo() (map[string]interface{}, error) {
	// Get current branch
	branch, _ := gs.execGit("branch", "--show-current")

	// Get remote URL
	remoteURL, _ := gs.execGit("remote", "get-url", "origin")

	// Get last commit
	lastCommit, _ := gs.execGit("log", "-1", "--oneline")

	return map[string]interface{}{
		"working_dir":    gs.workingDir,
		"current_branch": strings.TrimSpace(branch),
		"remote_url":     strings.TrimSpace(remoteURL),
		"last_commit":    strings.TrimSpace(lastCommit),
	}, nil
}

func (gs *GitService) getRemoteInfo() (map[string]interface{}, error) {
	remotes, err := gs.execGit("remote", "-v")
	if err != nil {
		return nil, err
	}

	var remoteList []map[string]string
	scanner := bufio.NewScanner(strings.NewReader(remotes))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			remoteList = append(remoteList, map[string]string{
				"name": parts[0],
				"url":  parts[1],
				"type": strings.Trim(parts[2], "()"),
			})
		}
	}

	return map[string]interface{}{
		"remotes": remoteList,
	}, nil
}

func (gs *GitService) handleWorkflowPrompt(args json.RawMessage) (interface{}, error) {
	// Get current status to suggest workflow
	status, err := gs.execGit("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	suggestions := []string{}

	if strings.Contains(status, "M ") {
		suggestions = append(suggestions, "You have staged changes ready to commit")
	}
	if strings.Contains(status, " M") {
		suggestions = append(suggestions, "You have unstaged changes - consider using git_add")
	}
	if strings.Contains(status, "??") {
		suggestions = append(suggestions, "You have untracked files - consider adding them")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Working tree is clean")
	}

	return map[string]interface{}{
		"workflow_suggestions": suggestions,
		"status":               status,
	}, nil
}

func (gs *GitService) handleCommitMessagePrompt(args json.RawMessage) (interface{}, error) {
	// Get staged changes
	diff, err := gs.execGit("diff", "--staged", "--name-only")
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(diff), "\n")
	if len(files) == 1 && files[0] == "" {
		return map[string]interface{}{
			"suggestion": "No staged changes to commit",
		}, nil
	}

	// Simple commit message suggestions based on changed files
	suggestion := "Update"
	if len(files) == 1 {
		suggestion = fmt.Sprintf("Update %s", filepath.Base(files[0]))
	} else {
		suggestion = fmt.Sprintf("Update %d files", len(files))
	}

	return map[string]interface{}{
		"suggestion":    suggestion,
		"changed_files": files,
	}, nil
}
