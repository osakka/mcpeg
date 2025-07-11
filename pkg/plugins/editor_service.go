package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/osakka/mcpeg/internal/registry"
)

// EditorService provides file system operations and editing capabilities
type EditorService struct {
	*BasePlugin
	workingDir string
	maxFileSize int64
	allowedExts map[string]bool
}

// NewEditorService creates a new editor service plugin
func NewEditorService() *EditorService {
	return &EditorService{
		BasePlugin: NewBasePlugin(
			"editor",
			"1.0.0",
			"File system operations and editing capabilities for development workflows",
		),
		maxFileSize: 10 * 1024 * 1024, // 10MB default limit
		allowedExts: map[string]bool{
			".go":   true,
			".js":   true,
			".ts":   true,
			".py":   true,
			".java": true,
			".cpp":  true,
			".c":    true,
			".h":    true,
			".md":   true,
			".txt":  true,
			".json": true,
			".yaml": true,
			".yml":  true,
			".xml":  true,
			".html": true,
			".css":  true,
			".sql":  true,
			".sh":   true,
			".env":  true,
		},
	}
}

// Initialize initializes the editor service
func (es *EditorService) Initialize(ctx context.Context, config PluginConfig) error {
	if err := es.BasePlugin.Initialize(ctx, config); err != nil {
		return err
	}
	
	// Set working directory
	es.workingDir = "."
	if configWorkDir, ok := config.Config["working_dir"].(string); ok {
		es.workingDir = configWorkDir
	}
	
	// Set max file size if configured
	if configMaxSize, ok := config.Config["max_file_size"].(float64); ok {
		es.maxFileSize = int64(configMaxSize)
	}
	
	// Set allowed extensions if configured
	if configExts, ok := config.Config["allowed_extensions"].([]interface{}); ok {
		es.allowedExts = make(map[string]bool)
		for _, ext := range configExts {
			if extStr, ok := ext.(string); ok {
				es.allowedExts[extStr] = true
			}
		}
	}
	
	es.logger.Info("editor_service_initialized",
		"working_dir", es.workingDir,
		"max_file_size", es.maxFileSize,
		"allowed_extensions", len(es.allowedExts))
	
	return nil
}

// GetTools returns the tools provided by the editor service
func (es *EditorService) GetTools() []registry.ToolDefinition {
	return []registry.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read the contents of a file",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
					"encoding": map[string]interface{}{
						"type":        "string",
						"description": "File encoding (default: utf-8)",
						"default":     "utf-8",
					},
					"lines": map[string]interface{}{
						"type":        "object",
						"description": "Read specific line range",
						"properties": map[string]interface{}{
							"start": map[string]interface{}{
								"type": "integer",
								"minimum": 1,
							},
							"end": map[string]interface{}{
								"type": "integer",
								"minimum": 1,
							},
						},
					},
				},
				"required": []string{"path"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Read entire file",
					Input: map[string]interface{}{
						"path": "src/main.go",
					},
				},
				{
					Description: "Read specific lines",
					Input: map[string]interface{}{
						"path": "src/main.go",
						"lines": map[string]interface{}{
							"start": 10,
							"end":   20,
						},
					},
				},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write to the file",
					},
					"encoding": map[string]interface{}{
						"type":        "string",
						"description": "File encoding (default: utf-8)",
						"default":     "utf-8",
					},
					"create_dirs": map[string]interface{}{
						"type":        "boolean",
						"description": "Create parent directories if they don't exist",
						"default":     false,
					},
					"backup": map[string]interface{}{
						"type":        "boolean",
						"description": "Create backup of existing file",
						"default":     false,
					},
				},
				"required": []string{"path", "content"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Write to file",
					Input: map[string]interface{}{
						"path":    "src/hello.go",
						"content": "package main\n\nfunc main() {\n    println(\"Hello, World!\")\n}",
					},
				},
			},
		},
		{
			Name:        "create_file",
			Description: "Create a new file with optional template content",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the new file",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Initial content for the file",
						"default":     "",
					},
					"template": map[string]interface{}{
						"type":        "string",
						"description": "Template to use for file creation",
					},
					"create_dirs": map[string]interface{}{
						"type":        "boolean",
						"description": "Create parent directories if they don't exist",
						"default":     true,
					},
				},
				"required": []string{"path"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Create empty file",
					Input: map[string]interface{}{
						"path": "docs/README.md",
					},
				},
			},
		},
		{
			Name:        "delete_file",
			Description: "Delete a file or directory",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file or directory to delete",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "Delete directories recursively",
						"default":     false,
					},
					"confirm": map[string]interface{}{
						"type":        "boolean",
						"description": "Confirmation required for deletion",
						"default":     false,
					},
				},
				"required": []string{"path", "confirm"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Delete a file",
					Input: map[string]interface{}{
						"path":    "temp/old_file.txt",
						"confirm": true,
					},
				},
			},
		},
		{
			Name:        "list_directory",
			Description: "List contents of a directory",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the directory to list",
						"default":     ".",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "List contents recursively",
						"default":     false,
					},
					"show_hidden": map[string]interface{}{
						"type":        "boolean",
						"description": "Include hidden files and directories",
						"default":     false,
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "File pattern to match (supports wildcards)",
					},
				},
			},
			Examples: []registry.ToolExample{
				{
					Description: "List current directory",
					Input: map[string]interface{}{
						"path": ".",
					},
				},
				{
					Description: "List Go files recursively",
					Input: map[string]interface{}{
						"path":      "src",
						"recursive": true,
						"pattern":   "*.go",
					},
				},
			},
		},
		{
			Name:        "search_files",
			Description: "Search for text within files",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Text pattern to search for",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to search in (default: current directory)",
						"default":     ".",
					},
					"file_pattern": map[string]interface{}{
						"type":        "string",
						"description": "File pattern to search within",
					},
					"case_sensitive": map[string]interface{}{
						"type":        "boolean",
						"description": "Case sensitive search",
						"default":     false,
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return",
						"default":     100,
						"minimum":     1,
						"maximum":     1000,
					},
				},
				"required": []string{"pattern"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Search for function definition",
					Input: map[string]interface{}{
						"pattern":      "func main",
						"file_pattern": "*.go",
					},
				},
			},
		},
		{
			Name:        "move_file",
			Description: "Move or rename a file or directory",
			Category:    "editor",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source path",
					},
					"destination": map[string]interface{}{
						"type":        "string",
						"description": "Destination path",
					},
					"create_dirs": map[string]interface{}{
						"type":        "boolean",
						"description": "Create destination directories if they don't exist",
						"default":     false,
					},
				},
				"required": []string{"source", "destination"},
			},
			Examples: []registry.ToolExample{
				{
					Description: "Rename a file",
					Input: map[string]interface{}{
						"source":      "old_name.go",
						"destination": "new_name.go",
					},
				},
			},
		},
	}
}

// GetResources returns resources provided by the editor service
func (es *EditorService) GetResources() []registry.ResourceDefinition {
	return []registry.ResourceDefinition{
		{
			Name:        "file_tree",
			Type:        "application/json",
			Description: "File system tree structure",
		},
		{
			Name:        "file_stats",
			Type:        "application/json",
			Description: "File system statistics and usage",
		},
	}
}

// GetPrompts returns prompts provided by the editor service
func (es *EditorService) GetPrompts() []registry.PromptDefinition {
	return []registry.PromptDefinition{
		{
			Name:        "code_review",
			Description: "Generate code review suggestions for files",
			Category:    "analysis",
		},
		{
			Name:        "file_summary",
			Description: "Generate summary of file contents",
			Category:    "analysis",
		},
	}
}

// CallTool executes an editor service tool
func (es *EditorService) CallTool(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	start := time.Now()
	defer func() {
		es.LogToolCall(name, time.Since(start), nil)
	}()
	
	switch name {
	case "read_file":
		return es.handleReadFile(args)
	case "write_file":
		return es.handleWriteFile(args)
	case "create_file":
		return es.handleCreateFile(args)
	case "delete_file":
		return es.handleDeleteFile(args)
	case "list_directory":
		return es.handleListDirectory(args)
	case "search_files":
		return es.handleSearchFiles(args)
	case "move_file":
		return es.handleMoveFile(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ReadResource reads an editor service resource
func (es *EditorService) ReadResource(ctx context.Context, uri string) (interface{}, error) {
	start := time.Now()
	defer func() {
		es.LogResourceAccess(uri, time.Since(start), nil)
	}()
	
	switch uri {
	case "file_tree":
		return es.getFileTree()
	case "file_stats":
		return es.getFileStats()
	default:
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}
}

// ListResources lists available resources
func (es *EditorService) ListResources(ctx context.Context) ([]registry.ResourceDefinition, error) {
	return es.GetResources(), nil
}

// GetPrompt returns a prompt
func (es *EditorService) GetPrompt(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	switch name {
	case "code_review":
		return es.handleCodeReviewPrompt(args)
	case "file_summary":
		return es.handleFileSummaryPrompt(args)
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

// Tool handlers

func (es *EditorService) handleReadFile(args json.RawMessage) (interface{}, error) {
	var req struct {
		Path     string `json:"path"`
		Encoding string `json:"encoding"`
		Lines    *struct {
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"lines,omitempty"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if req.Encoding == "" {
		req.Encoding = "utf-8"
	}
	
	// Validate file path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	// Check file size
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	
	if fileInfo.Size() > es.maxFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, max: %d)", fileInfo.Size(), es.maxFileSize)
	}
	
	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	contentStr := string(content)
	
	// Handle line range if specified
	if req.Lines != nil {
		lines := strings.Split(contentStr, "\n")
		start := req.Lines.Start - 1 // Convert to 0-based
		end := req.Lines.End
		
		if start < 0 {
			start = 0
		}
		if end > len(lines) {
			end = len(lines)
		}
		if start >= end {
			return nil, fmt.Errorf("invalid line range")
		}
		
		contentStr = strings.Join(lines[start:end], "\n")
	}
	
	es.metrics.Inc("editor_read_operations_total")
	es.metrics.Add("editor_bytes_read", float64(len(content)))
	
	return map[string]interface{}{
		"path":     req.Path,
		"content":  contentStr,
		"size":     len(content),
		"encoding": req.Encoding,
		"lines":    req.Lines,
	}, nil
}

func (es *EditorService) handleWriteFile(args json.RawMessage) (interface{}, error) {
	var req struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		Encoding   string `json:"encoding"`
		CreateDirs bool   `json:"create_dirs"`
		Backup     bool   `json:"backup"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if req.Encoding == "" {
		req.Encoding = "utf-8"
	}
	
	// Validate file path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	// Create directories if requested
	if req.CreateDirs {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directories: %w", err)
		}
	}
	
	// Create backup if requested and file exists
	var backupPath string
	if req.Backup {
		if _, err := os.Stat(fullPath); err == nil {
			backupPath = fullPath + ".backup." + time.Now().Format("20060102150405")
			if err := es.copyFile(fullPath, backupPath); err != nil {
				es.logger.Warn("failed_to_create_backup", "error", err)
			}
		}
	}
	
	// Write file
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	
	es.metrics.Inc("editor_write_operations_total")
	es.metrics.Add("editor_bytes_written", float64(len(req.Content)))
	
	return map[string]interface{}{
		"success":     true,
		"path":        req.Path,
		"size":        len(req.Content),
		"backup_path": backupPath,
	}, nil
}

func (es *EditorService) handleCreateFile(args json.RawMessage) (interface{}, error) {
	var req struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		Template   string `json:"template"`
		CreateDirs bool   `json:"create_dirs"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	// Validate file path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", req.Path)
	}
	
	// Apply template if specified
	content := req.Content
	if req.Template != "" {
		content = es.applyTemplate(req.Template, req.Path)
	}
	
	// Create directories if requested
	if req.CreateDirs {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directories: %w", err)
		}
	}
	
	// Create file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	
	es.metrics.Inc("editor_create_operations_total")
	
	return map[string]interface{}{
		"success":  true,
		"path":     req.Path,
		"size":     len(content),
		"template": req.Template,
	}, nil
}

func (es *EditorService) handleDeleteFile(args json.RawMessage) (interface{}, error) {
	var req struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
		Confirm   bool   `json:"confirm"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if !req.Confirm {
		return nil, fmt.Errorf("deletion requires confirmation")
	}
	
	// Validate file path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	// Check if path exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", req.Path)
	}
	
	var deletedItems int
	
	if fileInfo.IsDir() {
		if req.Recursive {
			err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
				if err == nil {
					deletedItems++
				}
				return err
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory: %w", err)
			}
			
			err = os.RemoveAll(fullPath)
		} else {
			err = os.Remove(fullPath)
			deletedItems = 1
		}
	} else {
		err = os.Remove(fullPath)
		deletedItems = 1
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to delete: %w", err)
	}
	
	es.metrics.Inc("editor_delete_operations_total")
	es.metrics.Add("editor_items_deleted", float64(deletedItems))
	
	return map[string]interface{}{
		"success":       true,
		"path":          req.Path,
		"deleted_items": deletedItems,
		"was_directory": fileInfo.IsDir(),
	}, nil
}

func (es *EditorService) handleListDirectory(args json.RawMessage) (interface{}, error) {
	var req struct {
		Path       string  `json:"path"`
		Recursive  bool    `json:"recursive"`
		ShowHidden bool    `json:"show_hidden"`
		Pattern    *string `json:"pattern,omitempty"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if req.Path == "" {
		req.Path = "."
	}
	
	// Validate path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	var items []map[string]interface{}
	
	if req.Recursive {
		err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			
			// Skip hidden files if not requested
			if !req.ShowHidden && strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			
			// Apply pattern filter
			if req.Pattern != nil && !es.matchPattern(d.Name(), *req.Pattern) {
				return nil
			}
			
			relPath, _ := filepath.Rel(es.workingDir, path)
			
			info, _ := d.Info()
			item := map[string]interface{}{
				"name":    d.Name(),
				"path":    relPath,
				"is_dir":  d.IsDir(),
				"size":    info.Size(),
				"mod_time": info.ModTime(),
			}
			
			items = append(items, item)
			return nil
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}
		
		for _, entry := range entries {
			// Skip hidden files if not requested
			if !req.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			
			// Apply pattern filter
			if req.Pattern != nil && !es.matchPattern(entry.Name(), *req.Pattern) {
				continue
			}
			
			info, _ := entry.Info()
			item := map[string]interface{}{
				"name":    entry.Name(),
				"path":    filepath.Join(req.Path, entry.Name()),
				"is_dir":  entry.IsDir(),
				"size":    info.Size(),
				"mod_time": info.ModTime(),
			}
			
			items = append(items, item)
		}
	}
	
	es.metrics.Inc("editor_list_operations_total")
	
	return map[string]interface{}{
		"path":      req.Path,
		"items":     items,
		"count":     len(items),
		"recursive": req.Recursive,
	}, nil
}

func (es *EditorService) handleSearchFiles(args json.RawMessage) (interface{}, error) {
	var req struct {
		Pattern       string  `json:"pattern"`
		Path          string  `json:"path"`
		FilePattern   *string `json:"file_pattern,omitempty"`
		CaseSensitive bool    `json:"case_sensitive"`
		MaxResults    int     `json:"max_results"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	if req.Path == "" {
		req.Path = "."
	}
	
	if req.MaxResults <= 0 {
		req.MaxResults = 100
	}
	
	// Validate path
	if err := es.validatePath(req.Path); err != nil {
		return nil, err
	}
	
	fullPath := filepath.Join(es.workingDir, req.Path)
	
	var results []map[string]interface{}
	searchPattern := req.Pattern
	if !req.CaseSensitive {
		searchPattern = strings.ToLower(searchPattern)
	}
	
	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and hidden files
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		
		// Apply file pattern filter
		if req.FilePattern != nil && !es.matchPattern(d.Name(), *req.FilePattern) {
			return nil
		}
		
		// Check if we've reached max results
		if len(results) >= req.MaxResults {
			return filepath.SkipAll
		}
		
		// Search within file
		matches, err := es.searchInFile(path, searchPattern, req.CaseSensitive)
		if err != nil {
			return nil // Skip files that can't be read
		}
		
		if len(matches) > 0 {
			relPath, _ := filepath.Rel(es.workingDir, path)
			results = append(results, map[string]interface{}{
				"file":    relPath,
				"matches": matches,
			})
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	es.metrics.Inc("editor_search_operations_total")
	
	return map[string]interface{}{
		"pattern":     req.Pattern,
		"results":     results,
		"result_count": len(results),
		"max_results": req.MaxResults,
	}, nil
}

func (es *EditorService) handleMoveFile(args json.RawMessage) (interface{}, error) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		CreateDirs  bool   `json:"create_dirs"`
	}
	
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}
	
	// Validate paths
	if err := es.validatePath(req.Source); err != nil {
		return nil, fmt.Errorf("invalid source path: %w", err)
	}
	if err := es.validatePath(req.Destination); err != nil {
		return nil, fmt.Errorf("invalid destination path: %w", err)
	}
	
	sourcePath := filepath.Join(es.workingDir, req.Source)
	destPath := filepath.Join(es.workingDir, req.Destination)
	
	// Check if source exists
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("source does not exist: %s", req.Source)
	}
	
	// Create destination directories if requested
	if req.CreateDirs {
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create destination directories: %w", err)
		}
	}
	
	// Move file
	if err := os.Rename(sourcePath, destPath); err != nil {
		return nil, fmt.Errorf("failed to move file: %w", err)
	}
	
	es.metrics.Inc("editor_move_operations_total")
	
	return map[string]interface{}{
		"success":     true,
		"source":      req.Source,
		"destination": req.Destination,
	}, nil
}

// Helper methods

func (es *EditorService) validatePath(path string) error {
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	
	// Check file extension if restrictions are in place
	if len(es.allowedExts) > 0 {
		ext := filepath.Ext(path)
		if ext != "" && !es.allowedExts[ext] {
			return fmt.Errorf("file extension not allowed: %s", ext)
		}
	}
	
	return nil
}

func (es *EditorService) matchPattern(name, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	
	// Basic wildcard support
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(name, prefix)
	}
	
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(name, suffix)
	}
	
	return name == pattern
}

func (es *EditorService) searchInFile(filePath, pattern string, caseSensitive bool) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var matches []map[string]interface{}
	scanner := bufio.NewScanner(file)
	lineNum := 1
	
	for scanner.Scan() {
		line := scanner.Text()
		searchLine := line
		
		if !caseSensitive {
			searchLine = strings.ToLower(line)
		}
		
		if strings.Contains(searchLine, pattern) {
			matches = append(matches, map[string]interface{}{
				"line_number": lineNum,
				"line":        line,
			})
		}
		
		lineNum++
	}
	
	return matches, scanner.Err()
}

func (es *EditorService) copyFile(src, dst string) error {
	sourceData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	
	return os.WriteFile(dst, sourceData, 0644)
}

func (es *EditorService) applyTemplate(template, path string) string {
	// Simple template system - could be enhanced
	switch template {
	case "go_main":
		return fmt.Sprintf("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n")
	case "go_package":
		packageName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return fmt.Sprintf("package %s\n\n// Package %s provides...\n", packageName, packageName)
	case "readme":
		return "# Project Title\n\nDescription of the project.\n\n## Usage\n\n```bash\n# Example usage\n```\n"
	default:
		return ""
	}
}

func (es *EditorService) getFileTree() (map[string]interface{}, error) {
	tree := make(map[string]interface{})
	
	err := filepath.WalkDir(es.workingDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip hidden files and directories
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		relPath, _ := filepath.Rel(es.workingDir, path)
		if relPath == "." {
			return nil
		}
		
		info, _ := d.Info()
		tree[relPath] = map[string]interface{}{
			"is_dir":  d.IsDir(),
			"size":    info.Size(),
			"mod_time": info.ModTime(),
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"working_dir": es.workingDir,
		"tree":        tree,
	}, nil
}

func (es *EditorService) getFileStats() (map[string]interface{}, error) {
	var totalFiles, totalDirs int
	var totalSize int64
	
	err := filepath.WalkDir(es.workingDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			totalDirs++
		} else {
			totalFiles++
			if info, err := d.Info(); err == nil {
				totalSize += info.Size()
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"total_files":      totalFiles,
		"total_directories": totalDirs,
		"total_size_bytes": totalSize,
		"working_dir":      es.workingDir,
	}, nil
}

func (es *EditorService) handleCodeReviewPrompt(args json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"prompt": "Analyze code for potential improvements",
		"suggestions": []string{
			"Check for code style consistency",
			"Look for potential bugs or edge cases",
			"Review error handling",
			"Consider performance optimizations",
		},
	}, nil
}

func (es *EditorService) handleFileSummaryPrompt(args json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"prompt": "Generate a summary of file contents",
		"instructions": "Read the file and provide a concise summary of its purpose and functionality",
	}, nil
}