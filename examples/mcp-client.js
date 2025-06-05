// Example MCP client for Beagle Run
// This demonstrates how to integrate with the MCP server

const API_BASE = 'http://localhost:7777/mcp';

class BeagleRunClient {
  constructor(clientName = 'example-client') {
    this.clientName = clientName;
    this.clientId = null;
    this.eventSource = null;
  }

  async connect() {
    const response = await fetch(`${API_BASE}/connect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ clientName: this.clientName })
    });

    const data = await response.json();
    this.clientId = data.clientId;
    console.log('Connected with client ID:', this.clientId);
    console.log('Available capabilities:', data.capabilities);
    
    return this.clientId;
  }

  async getScripts() {
    const response = await fetch(`${API_BASE}/scripts`);
    return response.json();
  }

  async getProcesses() {
    const response = await fetch(`${API_BASE}/processes`);
    return response.json();
  }

  async getLogs(processId = null) {
    const url = processId 
      ? `${API_BASE}/logs?processId=${processId}`
      : `${API_BASE}/logs`;
    const response = await fetch(url);
    return response.json();
  }

  async executeScript(scriptName) {
    const response = await fetch(`${API_BASE}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ script: scriptName })
    });
    return response.json();
  }

  async stopProcess(processId) {
    const response = await fetch(`${API_BASE}/stop`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ processId })
    });
    return response.json();
  }

  async searchLogs(query) {
    const response = await fetch(`${API_BASE}/search?query=${encodeURIComponent(query)}`);
    return response.json();
  }

  subscribeToEvents(onEvent) {
    if (!this.clientId) {
      throw new Error('Must connect first');
    }

    this.eventSource = new EventSource(`${API_BASE}/events?clientId=${this.clientId}`);
    
    this.eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      onEvent(data);
    };

    this.eventSource.onerror = (error) => {
      console.error('EventSource error:', error);
    };
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }
}

// Example usage
async function main() {
  const client = new BeagleRunClient('example-app');
  
  try {
    // Connect to the server
    await client.connect();
    
    // Get available scripts
    const scripts = await client.getScripts();
    console.log('Available scripts:', scripts);
    
    // Subscribe to events
    client.subscribeToEvents((event) => {
      console.log('Event received:', event.type);
      
      switch (event.type) {
        case 'error.detected':
          console.error('Error detected:', event.data);
          break;
        case 'test.failed':
          console.error('Test failed:', event.data);
          break;
        case 'build.event':
          console.log('Build event:', event.data);
          break;
      }
    });
    
    // Execute a script
    if (scripts.dev) {
      console.log('Starting dev script...');
      const process = await client.executeScript('dev');
      console.log('Started process:', process);
      
      // Wait a bit then check logs
      setTimeout(async () => {
        const logs = await client.getLogs(process.ID);
        console.log(`Logs for ${process.Name}:`, logs.slice(-5));
      }, 5000);
    }
    
  } catch (error) {
    console.error('Error:', error);
  }
}

// Run the example
if (require.main === module) {
  main().catch(console.error);
}

module.exports = BeagleRunClient;