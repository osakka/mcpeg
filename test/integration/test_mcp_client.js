#!/usr/bin/env node

/**
 * Simple MCP Test Client for MCpeg Gateway
 * Tests the MCP JSON-RPC endpoints
 */

const http = require('http');

class MCPTestClient {
    constructor(baseUrl = 'http://localhost:8080') {
        this.baseUrl = baseUrl;
        this.requestId = 1;
    }

    async makeRequest(method, params = {}) {
        const payload = {
            jsonrpc: '2.0',
            id: this.requestId++,
            method: method,
            params: params
        };

        return new Promise((resolve, reject) => {
            const data = JSON.stringify(payload);
            const options = {
                hostname: 'localhost',
                port: 8080,
                path: '/mcp',
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Content-Length': Buffer.byteLength(data)
                }
            };

            const req = http.request(options, (res) => {
                let body = '';
                res.on('data', (chunk) => {
                    body += chunk;
                });
                res.on('end', () => {
                    try {
                        const response = JSON.parse(body);
                        resolve(response);
                    } catch (error) {
                        reject(new Error(`Failed to parse response: ${body}`));
                    }
                });
            });

            req.on('error', (error) => {
                reject(error);
            });

            req.write(data);
            req.end();
        });
    }

    async testToolsList() {
        console.log('🔧 Testing tools/list...');
        const response = await this.makeRequest('tools/list');
        if (response.error) {
            console.error('❌ Error:', response.error.message);
            return false;
        }
        console.log(`✅ Found ${response.result.tools.length} tools`);
        response.result.tools.slice(0, 3).forEach(tool => {
            console.log(`   - ${tool.name}: ${tool.description}`);
        });
        if (response.result.tools.length > 3) {
            console.log(`   ... and ${response.result.tools.length - 3} more`);
        }
        return true;
    }

    async testResourcesList() {
        console.log('📚 Testing resources/list...');
        const response = await this.makeRequest('resources/list');
        if (response.error) {
            console.error('❌ Error:', response.error.message);
            return false;
        }
        console.log(`✅ Found ${response.result.resources.length} resources`);
        response.result.resources.forEach(resource => {
            console.log(`   - ${resource.name}: ${resource.description}`);
        });
        return true;
    }

    async testPromptsList() {
        console.log('💭 Testing prompts/list...');
        const response = await this.makeRequest('prompts/list');
        if (response.error) {
            console.error('❌ Error:', response.error.message);
            return false;
        }
        console.log(`✅ Found ${response.result.prompts.length} prompts`);
        response.result.prompts.forEach(prompt => {
            console.log(`   - ${prompt.name}: ${prompt.description}`);
        });
        return true;
    }

    async testToolCall() {
        console.log('🛠️  Testing tools/call with memory.memory_list...');
        const response = await this.makeRequest('tools/call', {
            name: 'memory.memory_list',
            arguments: {}
        });
        if (response.error) {
            console.error('❌ Error:', response.error.message);
            return false;
        }
        console.log('✅ Tool call successful');
        console.log(`   Result: ${response.result.content[0].text}`);
        return true;
    }

    async testEditorTool() {
        console.log('📝 Testing tools/call with editor.list_directory...');
        const response = await this.makeRequest('tools/call', {
            name: 'editor.list_directory',
            arguments: { path: '.' }
        });
        if (response.error) {
            console.error('❌ Error:', response.error.message);
            return false;
        }
        console.log('✅ Editor tool call successful');
        console.log(`   Found files in current directory`);
        return true;
    }

    async runAllTests() {
        console.log('🚀 Starting MCpeg MCP Server Tests\n');
        
        const tests = [
            () => this.testToolsList(),
            () => this.testResourcesList(), 
            () => this.testPromptsList(),
            () => this.testToolCall(),
            () => this.testEditorTool()
        ];

        let passed = 0;
        let failed = 0;

        for (const test of tests) {
            try {
                const result = await test();
                if (result) {
                    passed++;
                } else {
                    failed++;
                }
            } catch (error) {
                console.error('❌ Test failed with exception:', error.message);
                failed++;
            }
            console.log(''); // Empty line between tests
        }

        console.log('📊 Test Results:');
        console.log(`   ✅ Passed: ${passed}`);
        console.log(`   ❌ Failed: ${failed}`);
        console.log(`   📈 Success rate: ${((passed / (passed + failed)) * 100).toFixed(1)}%`);

        return failed === 0;
    }
}

// Run tests if this file is executed directly
if (require.main === module) {
    const client = new MCPTestClient();
    client.runAllTests()
        .then(success => {
            process.exit(success ? 0 : 1);
        })
        .catch(error => {
            console.error('Fatal error:', error);
            process.exit(1);
        });
}

module.exports = MCPTestClient;