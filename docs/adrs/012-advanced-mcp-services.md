# ADR-012: Advanced MCP Services (Git, Editor, Search, Memory)

## Status

**ACCEPTED** - *2025-07-11*

## Context

Beyond basic database and API integrations, MCPEG can provide sophisticated services that leverage the full power of the MCP protocol. These advanced services can transform how LLMs interact with development environments and persistent memory.

Advanced use cases identified:
1. **Git Service**: Complete repository management through MCP
2. **Editor Service**: Full-featured code editing capabilities
3. **Search Service**: Semantic and contextual search across all data
4. **Memory Service**: LLM memory enhancement and evolution

## Decision

We will implement these advanced services as first-class MCP adapters, each providing sophisticated tools, resources, and prompts that go far beyond simple API calls.

## Advanced Service Designs

### 1. Git Service - Complete Repository Control

**MCP Tools:**
- `git_status` - Get repository status with context
- `git_log` - Repository history with semantic analysis
- `git_diff` - Smart diff with change explanations
- `git_commit` - Intelligent commit with auto-generated messages
- `git_branch` - Branch management with merge conflict prediction
- `git_blame` - Code attribution with context
- `git_search` - Search across commit history and content
- `git_analyze` - Repository analysis (hotspots, dependencies)

**MCP Resources:**
- `git://repo/status` - Current repository state
- `git://repo/history/{branch}` - Branch history
- `git://repo/file/{path}@{commit}` - File at specific commit
- `git://repo/conflicts` - Merge conflicts with resolution suggestions
- `git://repo/analytics` - Repository analytics and insights

**MCP Prompts:**
- `commit_message_generator` - Generate semantic commit messages
- `code_review` - Automated code review based on changes
- `merge_strategy` - Suggest merge strategies for complex scenarios
- `repository_health` - Analyze repository health and suggest improvements

**Configuration Example:**
```yaml
# services/git.yaml
type: "vcs"
driver: "git"

repositories:
  - path: "/workspace/project"
    name: "main_project"
    permissions:
      read: true
      write: true
      push: false  # Safety for production

ai_features:
  commit_analysis: true
  conflict_resolution: true
  code_quality_checks: true
  semantic_search: true

safety:
  require_confirmation: ["push", "force-push", "rebase"]
  protected_branches: ["main", "production"]
  max_files_per_commit: 50
```

### 2. Editor Service - Advanced Code Manipulation

**MCP Tools:**
- `read_file` - Read file with syntax highlighting and analysis
- `write_file` - Write file with validation and formatting
- `edit_range` - Precise range-based editing
- `refactor_code` - Automated refactoring operations
- `find_references` - Find all references to symbols
- `auto_complete` - Context-aware code completion
- `format_code` - Format code according to language standards
- `lint_code` - Lint code and suggest fixes

**MCP Resources:**
- `editor://file/{path}` - File content with metadata
- `editor://project/symbols` - All symbols in project
- `editor://diagnostics` - Current errors and warnings
- `editor://workspace/files` - Workspace file tree

**MCP Prompts:**
- `code_explanation` - Explain code functionality
- `bug_finder` - Identify potential bugs
- `optimization_suggestions` - Performance improvement suggestions
- `architecture_review` - High-level architecture analysis

### 3. Search Service - Semantic Multi-Modal Search

**MCP Tools:**
- `semantic_search` - AI-powered semantic search across all data
- `vector_search` - Vector similarity search
- `hybrid_search` - Combine keyword + semantic + vector search
- `search_analyze` - Analyze search patterns and results
- `index_content` - Index new content for search
- `search_suggest` - Search query suggestions

**MCP Resources:**
- `search://index/stats` - Search index statistics
- `search://results/{query}` - Cached search results
- `search://suggestions/{partial}` - Query suggestions

**Search Across All MCPEG Services:**
```yaml
# services/search.yaml
type: "search"
driver: "semantic"

# Index all MCPEG-accessible data
data_sources:
  - service: "mysql"
    tables: ["users", "orders", "products"]
    fields: ["name", "description", "content"]
  
  - service: "git"
    content: ["commits", "files", "issues"]
    
  - service: "editor"
    content: ["code", "comments", "documentation"]
    
  - service: "memory"
    content: ["conversations", "insights", "learnings"]

embeddings:
  model: "text-embedding-3-large"
  dimensions: 1536
  chunk_size: 1000
  overlap: 200

vector_store:
  type: "chroma"  # or "pinecone", "weaviate"
  collection: "mcpeg_knowledge"
```

### 4. Memory Service - LLM Memory Enhancement

**The Most Interesting Use Case!**

This service provides persistent, evolving memory for LLMs across conversations and time.

**MCP Tools:**
- `store_memory` - Store conversation insights and learnings
- `recall_memory` - Retrieve relevant memories based on context
- `update_memory` - Update existing memories with new information
- `forget_memory` - Remove outdated or incorrect memories
- `memory_search` - Search through all stored memories
- `memory_analytics` - Analyze memory patterns and evolution
- `memory_consolidate` - Merge and consolidate related memories
- `memory_export` - Export memories for backup or migration

**MCP Resources:**
- `memory://personal/{user}` - Personal memories for specific user
- `memory://contextual/{topic}` - Memories related to specific topics
- `memory://temporal/{timeframe}` - Memories from specific time periods
- `memory://insights/{domain}` - Domain-specific insights and learnings
- `memory://evolution/{concept}` - How understanding of concepts evolved

**Memory Structure:**
```json
{
  "id": "mem_001",
  "type": "insight",
  "content": "Claude prefers structured logging with complete context",
  "context": {
    "conversation_id": "conv_123",
    "timestamp": "2025-07-11T10:30:00Z",
    "participants": ["human", "claude"],
    "topic": "logging_design"
  },
  "metadata": {
    "confidence": 0.95,
    "importance": 8,
    "tags": ["development", "logging", "best_practices"],
    "related_memories": ["mem_045", "mem_067"]
  },
  "evolution": [
    {
      "timestamp": "2025-07-11T10:30:00Z",
      "action": "created",
      "trigger": "explicit_statement"
    }
  ]
}
```

**Memory Configuration:**
```yaml
# services/memory.yaml
type: "memory"
driver: "enhanced_llm_memory"

storage:
  primary: "vector_db"      # Vector storage for semantic search
  metadata: "sqlite"        # Structured metadata storage
  backup: "file_system"     # Periodic backups

memory_types:
  - name: "factual"
    description: "Factual information learned"
    confidence_threshold: 0.8
    importance_weight: 0.6
    
  - name: "preference"
    description: "User preferences and patterns"
    confidence_threshold: 0.7
    importance_weight: 0.9
    
  - name: "insight"
    description: "Insights and understanding"
    confidence_threshold: 0.9
    importance_weight: 0.8

evolution_rules:
  - trigger: "contradiction"
    action: "flag_for_review"
    
  - trigger: "reinforcement"
    action: "increase_confidence"
    
  - trigger: "temporal_decay"
    action: "decrease_importance"
    params:
      half_life: "30d"

privacy:
  encryption: true
  user_isolation: true
  retention_policy: "user_controlled"
  export_rights: true
```

**Memory Prompts:**
- `memory_reflection` - Reflect on what was learned in conversation
- `memory_connection` - Find connections between memories
- `memory_evolution` - Analyze how understanding has evolved
- `memory_summary` - Summarize memories for specific context

## Implementation Architecture

### Memory Service Deep Dive

**Core Components:**
1. **Memory Encoder**: Convert conversations into structured memories
2. **Memory Retriever**: Find relevant memories for current context
3. **Memory Evolver**: Update memories based on new information
4. **Memory Analyzer**: Analyze patterns and evolution

**Memory Evolution Algorithm:**
```go
type MemoryEvolution struct {
    Original    Memory
    Updates     []MemoryUpdate
    Confidence  float64
    Importance  float64
    LastAccess  time.Time
    RelatedIDs  []string
}

func (me *MemoryEvolution) Evolve(newInfo Information, context Context) {
    if me.Contradicts(newInfo) {
        me.FlagForReview(newInfo, context)
    } else if me.Reinforces(newInfo) {
        me.IncreaseConfidence(0.1)
    }
    
    me.UpdateConnections(context)
    me.RecalculateImportance()
}
```

## Consequences

### Positive
- **Revolutionary LLM Capabilities**: Memory service enables persistent learning
- **Complete Development Environment**: Git + Editor + Search covers full workflow
- **Compound Value**: Services enhance each other (Git search uses Memory insights)
- **Unique Differentiation**: No other MCP gateway provides this level of sophistication

### Negative
- **Complexity**: Significant implementation complexity
- **Resource Requirements**: Memory and search require substantial storage
- **Privacy Concerns**: Memory service needs careful privacy design
- **Performance**: Advanced features may impact response times

### Neutral
- **User Adoption Curve**: Advanced features require user education
- **Configuration Complexity**: Many options to configure correctly

## Implementation Priority

**Phase 1: Foundation** (Current)
- Basic adapters (MySQL, Weather, Scripts)
- Core infrastructure

**Phase 2: Development Tools**
- Git service
- Editor service
- Basic search

**Phase 3: Intelligence Layer**
- Memory service
- Advanced search with cross-service indexing
- AI-powered analysis and insights

## Memory Service Evolution Scenarios

1. **Learning Preferences**: "User prefers detailed explanations" → Adjust response style
2. **Technical Evolution**: "Initially thought X, but now understand Y" → Knowledge refinement
3. **Pattern Recognition**: "User always asks about security first" → Proactive security analysis
4. **Contextual Adaptation**: "In this project, use these specific patterns" → Context-aware assistance

This memory service could revolutionize how LLMs work with humans over time!