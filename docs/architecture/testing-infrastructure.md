# MCpeg Testing Infrastructure

## Overview

MCpeg implements a comprehensive testing infrastructure to ensure production-ready reliability of the MCP gateway and plugin ecosystem. The testing approach follows the XVC methodology with single source of truth, surgical precision, and zero-regression validation.

## Testing Architecture

### 1. Test Organization

```
test/
└── integration/                    # Integration test suite
    ├── test_mcp_client.js         # Automated MCP test client
    └── mcp-inspector-config.json  # MCP Inspector configuration
```

### 2. Documentation Structure

```
docs/testing/
└── mcp-testing-methodology.md     # Comprehensive testing methodology
```

## Test Components

### Automated MCP Test Client (`test/integration/test_mcp_client.js`)

**Purpose**: Comprehensive validation of MCP JSON-RPC protocol compliance and functionality.

**Capabilities**:
- Full MCP protocol validation (JSON-RPC 2.0)
- Tool discovery and execution testing
- Resource enumeration validation
- Prompt availability verification
- Error handling validation

**Test Coverage**:
```javascript
✅ Tools Discovery: 20 tools across 3 plugins
✅ Resources Discovery: 6 resources across 3 plugins  
✅ Prompts Discovery: 6 prompts across 3 plugins
✅ Tool Execution: Memory and Editor plugin validation
✅ Protocol Compliance: JSON-RPC 2.0 specification adherence
```

**Usage**:
```bash
# Run automated test suite
node test/integration/test_mcp_client.js

# Expected output: 100% success rate
📊 Test Results:
   ✅ Passed: 5
   ❌ Failed: 0
   📈 Success rate: 100.0%
```

### MCP Inspector Configuration (`test/integration/mcp-inspector-config.json`)

**Purpose**: Interactive testing and debugging of MCP server functionality.

**Configuration**:
```json
{
  "mcpServers": {
    "mcpeg-gateway": {
      "command": "./build/mcpeg",
      "args": ["gateway", "--dev"],
      "env": {
        "LOG_LEVEL": "debug",
        "MCP_DEBUG": "true"
      }
    }
  }
}
```

**Usage**:
```bash
# Start MCP Inspector with MCpeg
npx @modelcontextprotocol/inspector \
  --config test/integration/mcp-inspector-config.json \
  --server mcpeg-gateway
```

## Testing Methodology

### 1. Continuous Integration Testing

**Pre-commit Validation**:
```bash
# Build verification
./scripts/build.sh build

# Automated test execution
node test/integration/test_mcp_client.js

# Health verification
curl http://localhost:8080/health
```

**Integration Pipeline**:
1. Start MCpeg server in development mode
2. Wait for plugin initialization completion
3. Execute automated test suite
4. Verify 100% success rate
5. Validate Phase 2 discovery completion

### 2. Manual Testing Procedures

**Direct HTTP Testing**:
```bash
# Test tool discovery
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}'

# Test tool execution
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "memory.memory_list", "arguments": {}}}'

# Test resource discovery
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 3, "method": "resources/list", "params": {}}'
```

### 3. Performance Validation

**Response Time Benchmarks** (Local Testing):
- `tools/list`: ~10-50ms
- `tools/call`: ~50-200ms (tool dependent)
- `resources/list`: ~5-20ms
- `prompts/list`: ~5-20ms

**Concurrency Testing**:
- Thread-safe capability analysis validation
- Concurrent request handling verification
- Phase 2 discovery system stability

## Test Coverage Matrix

| Component | Test Type | Coverage | Status |
|-----------|-----------|----------|---------|
| **MCP Protocol** | Automated | JSON-RPC 2.0 compliance | ✅ Passing |
| **Tool Discovery** | Automated | 20 tools (Memory, Git, Editor) | ✅ Passing |
| **Tool Execution** | Automated | Plugin tool invocation | ✅ Passing |
| **Resource Discovery** | Automated | 6 resources across plugins | ✅ Passing |
| **Prompt Discovery** | Automated | 6 prompts across plugins | ✅ Passing |
| **Error Handling** | Manual | Invalid requests, edge cases | ✅ Passing |
| **Concurrency** | Automated | Thread-safe operations | ✅ Fixed |
| **Phase 2 Intelligence** | Automated | Discovery system stability | ✅ Passing |

## Validated Capabilities

### Memory Plugin (5 tools, 2 resources, 2 prompts)
```
Tools: memory_store, memory_retrieve, memory_list, memory_delete, memory_clear
Resources: memory_stats, memory_dump
Prompts: memory_search, memory_context
```

### Git Plugin (8 tools, 2 resources, 2 prompts)
```
Tools: git_status, git_diff, git_add, git_commit, git_push, git_pull, git_branch, git_log
Resources: git_repo_info, git_remote_info
Prompts: git_workflow, commit_message
```

### Editor Plugin (7 tools, 2 resources, 2 prompts)
```
Tools: read_file, write_file, create_file, delete_file, list_directory, search_files, move_file
Resources: file_tree, file_stats
Prompts: code_review, file_summary
```

## Quality Assurance

### Thread Safety Validation
- **Issue**: Concurrent map writes in Phase 2 capability analysis
- **Solution**: RWMutex synchronization in AnalysisEngine
- **Validation**: Zero race conditions under concurrent load

### Protocol Compliance
- **Standard**: MCP 2025-03-26 specification
- **Validation**: Full JSON-RPC 2.0 compliance
- **Coverage**: All method types (tools, resources, prompts)

### Error Handling
- **Invalid Requests**: Proper error responses
- **Timeout Handling**: Graceful degradation
- **Plugin Failures**: Isolated error handling

## Troubleshooting

### Common Issues

**Port Conflicts**:
```bash
# Stop existing instances
pkill -f "mcpeg gateway"

# Verify port availability
lsof -ti :8080
```

**Plugin Initialization Failures**:
```bash
# Check server logs
tail -f build/logs/mcpeg.log

# Verify plugin status
curl http://localhost:8080/admin/services | jq
```

**Test Client Failures**:
```bash
# Verify server is running
curl http://localhost:8080/health

# Check MCP endpoint
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}'
```

## Future Enhancements

### Planned Testing Improvements
1. **Load Testing**: Automated performance validation under load
2. **Chaos Testing**: Fault injection and recovery validation
3. **Security Testing**: Authentication and authorization validation
4. **Integration Testing**: External MCP client compatibility
5. **Regression Testing**: Automated regression detection

### Testing Infrastructure Evolution
- **CI/CD Integration**: Automated testing in deployment pipeline
- **Performance Monitoring**: Continuous performance regression detection
- **Coverage Analysis**: Code coverage tracking and reporting
- **Test Data Management**: Standardized test data sets

This testing infrastructure ensures MCpeg maintains production-grade reliability while supporting rapid development and deployment cycles.