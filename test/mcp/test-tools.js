#!/usr/bin/env node

// Test client for Brummer MCP server
const http = require('http');

const MCP_BASE = 'http://localhost:7777/mcp';

class MCPClient {
  async request(method, params = {}) {
    const message = {
      jsonrpc: '2.0',
      id: Date.now(),
      method,
      params
    };

    return new Promise((resolve, reject) => {
      const data = JSON.stringify(message);
      
      const options = {
        hostname: 'localhost',
        port: 7777,
        path: '/mcp',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': data.length
        }
      };

      const req = http.request(options, (res) => {
        let body = '';
        res.on('data', chunk => body += chunk);
        res.on('end', () => {
          try {
            const response = JSON.parse(body);
            if (response.error) {
              reject(new Error(`${response.error.message} (${response.error.code})`));
            } else {
              resolve(response.result);
            }
          } catch (e) {
            reject(e);
          }
        });
      });

      req.on('error', reject);
      req.write(data);
      req.end();
    });
  }

  async initialize() {
    console.log('🔌 Initializing MCP connection...');
    const result = await this.request('initialize');
    console.log('✅ Connected to:', result.serverInfo.name, result.serverInfo.version);
    console.log('📋 Capabilities:', Object.keys(result.capabilities).join(', '));
    return result;
  }

  async listTools() {
    console.log('\n🔧 Listing available tools...');
    const result = await this.request('tools/list');
    console.log(`Found ${result.tools.length} tools:`);
    
    for (const tool of result.tools) {
      console.log(`\n  📌 ${tool.name}`);
      const desc = tool.description.split('\n')[0];
      console.log(`     ${desc}`);
    }
    
    return result.tools;
  }

  async listScripts() {
    console.log('\n📜 Getting available scripts...');
    const result = await this.request('tools/call', {
      name: 'scripts/list',
      arguments: {}
    });
    
    const scripts = JSON.parse(result.content[0].text);
    console.log('Scripts:', scripts);
    return scripts;
  }

  async runScript(name) {
    console.log(`\n🚀 Starting script: ${name}`);
    const result = await this.request('tools/call', {
      name: 'scripts/run',
      arguments: { name }
    });
    
    const data = JSON.parse(result.content[0].text);
    console.log('Started:', data);
    return data;
  }

  async getStatus() {
    console.log('\n📊 Getting process status...');
    const result = await this.request('tools/call', {
      name: 'scripts/status',
      arguments: {}
    });
    
    const status = JSON.parse(result.content[0].text);
    console.log('Status:', status);
    return status;
  }

  async getLogs(limit = 20) {
    console.log(`\n📄 Getting last ${limit} log entries...`);
    const result = await this.request('tools/call', {
      name: 'logs/stream',
      arguments: { follow: false, limit }
    });
    
    const logs = JSON.parse(result.content[0].text);
    console.log(`Retrieved ${logs.length} log entries`);
    return logs;
  }

  async openBrowser(url) {
    console.log(`\n🌐 Opening browser: ${url}`);
    const result = await this.request('tools/call', {
      name: 'browser/open',
      arguments: { url }
    });
    
    const data = JSON.parse(result.content[0].text);
    console.log('Browser opened:', data);
    return data;
  }

  async listResources() {
    console.log('\n📚 Listing available resources...');
    const result = await this.request('resources/list');
    console.log(`Found ${result.resources.length} resources:`);
    
    for (const resource of result.resources) {
      console.log(`  📁 ${resource.uri} - ${resource.description}`);
    }
    
    return result.resources;
  }

  async readResource(uri) {
    console.log(`\n📖 Reading resource: ${uri}`);
    const result = await this.request('resources/read', { uri });
    
    const content = JSON.parse(result.contents[0].text);
    console.log(`Retrieved ${Array.isArray(content) ? content.length : 1} items`);
    return content;
  }
}

async function runTests() {
  const client = new MCPClient();
  
  try {
    // Initialize connection
    await client.initialize();
    
    // List available tools
    await client.listTools();
    
    // List resources
    await client.listResources();
    
    // Get available scripts
    await client.listScripts();
    
    // Get current status
    await client.getStatus();
    
    // Get some logs
    await client.getLogs(10);
    
    // Read a resource
    await client.readResource('scripts://available');
    
    // Example: Run a script (commented out to avoid actually starting processes)
    // await client.runScript('dev');
    
    // Example: Open browser (commented out to avoid opening browser)
    // await client.openBrowser('http://localhost:3000');
    
    console.log('\n✅ All tests completed successfully!');
    
  } catch (error) {
    console.error('\n❌ Test failed:', error.message);
    process.exit(1);
  }
}

// Run tests if called directly
if (require.main === module) {
  runTests();
}

module.exports = { MCPClient };