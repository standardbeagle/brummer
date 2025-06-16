#!/usr/bin/env node

// Test MCP Streamable HTTP Transport (SSE)
// Usage: node test-mcp-sse.js

const EventSource = require('eventsource');
const fetch = require('node-fetch');

const MCP_URL = 'http://localhost:7777/mcp';
const SESSION_ID = `test-session-${Date.now()}`;

console.log('Testing MCP Streamable HTTP Transport');
console.log('Session ID:', SESSION_ID);

// 1. Establish SSE connection for streaming
const eventSource = new EventSource(MCP_URL, {
  headers: {
    'Accept': 'text/event-stream',
    'Mcp-Session-Id': SESSION_ID
  }
});

eventSource.onopen = () => {
  console.log('SSE connection established');
};

eventSource.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Message:', JSON.stringify(msg, null, 2));
};

eventSource.onerror = (error) => {
  console.error('SSE error:', error);
};

// 2. Send requests via POST
async function sendRequest(method, params = {}, id = null) {
  const request = {
    jsonrpc: '2.0',
    method,
    params
  };
  
  if (id !== null) {
    request.id = id;
  }

  try {
    const response = await fetch(MCP_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'Mcp-Session-Id': SESSION_ID
      },
      body: JSON.stringify(request)
    });

    const result = await response.json();
    console.log(`Response for ${method}:`, JSON.stringify(result, null, 2));
    return result;
  } catch (error) {
    console.error(`Error sending ${method}:`, error);
  }
}

// 3. Test sequence
async function runTests() {
  // Initialize
  await sendRequest('initialize', {
    protocolVersion: '2024-11-05',
    capabilities: {}
  }, 1);

  // List tools
  await sendRequest('tools/list', {}, 2);

  // Subscribe to resources
  await sendRequest('resources/subscribe', {
    uri: 'logs://recent'
  }, 3);

  await sendRequest('resources/subscribe', {
    uri: 'processes://active'
  }, 4);

  // List resources
  await sendRequest('resources/list', {}, 5);

  // Read a resource
  await sendRequest('resources/read', {
    uri: 'logs://recent'
  }, 6);
}

// Run tests after a short delay
setTimeout(runTests, 1000);

// Keep the script running
process.on('SIGINT', () => {
  console.log('\nClosing connection...');
  eventSource.close();
  process.exit(0);
});