const http = require('http');

// MCP client configuration
const MCP_HOST = 'localhost';
const MCP_PORT = 7777;

// Helper function to make HTTP requests
function makeRequest(path, method = 'GET', data = null) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: MCP_HOST,
      port: MCP_PORT,
      path: path,
      method: method,
      headers: {
        'Content-Type': 'application/json'
      }
    };

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(body));
        } catch (e) {
          resolve(body);
        }
      });
    });

    req.on('error', reject);
    
    if (data) {
      req.write(JSON.stringify(data));
    }
    
    req.end();
  });
}

// Get telemetry sessions
async function getTelemetrySessions(processName = null) {
  const path = processName 
    ? `/telemetry/sessions?process=${encodeURIComponent(processName)}`
    : '/telemetry/sessions';
  
  try {
    const response = await makeRequest(path);
    return response;
  } catch (error) {
    console.error('Failed to get telemetry sessions:', error);
    return null;
  }
}

// Get specific session details
async function getTelemetrySession(sessionId) {
  try {
    const response = await makeRequest(`/telemetry/sessions/${sessionId}`);
    return response;
  } catch (error) {
    console.error('Failed to get telemetry session:', error);
    return null;
  }
}

// Monitor telemetry events in real-time
function monitorTelemetryEvents() {
  console.log('Monitoring telemetry events (SSE)...');
  
  const req = http.get(`http://${MCP_HOST}:${MCP_PORT}/events`, (res) => {
    res.on('data', (chunk) => {
      const lines = chunk.toString().split('\n');
      lines.forEach(line => {
        if (line.startsWith('data: ')) {
          try {
            const event = JSON.parse(line.substring(6));
            if (event.type === 'telemetry.received') {
              console.log('Telemetry event:', event);
            }
          } catch (e) {
            // Ignore parse errors
          }
        }
      });
    });
  });

  req.on('error', (err) => {
    console.error('SSE connection error:', err);
  });
}

// Main demo function
async function main() {
  console.log('Brummer Telemetry Client Demo');
  console.log('==============================\n');
  
  // Get all telemetry sessions
  console.log('Fetching all telemetry sessions...');
  const sessions = await getTelemetrySessions();
  
  if (sessions && sessions.length > 0) {
    console.log(`Found ${sessions.length} sessions:\n`);
    
    sessions.forEach((session, index) => {
      console.log(`Session ${index + 1}:`);
      console.log(`  ID: ${session.sessionId}`);
      console.log(`  URL: ${session.url}`);
      console.log(`  Process: ${session.processName}`);
      console.log(`  Duration: ${session.duration}s`);
      console.log(`  Events: ${session.eventCount}`);
      console.log(`  Errors: ${session.errorCount}`);
      console.log(`  Interactions: ${session.interactionCount}`);
      
      // Show performance metrics if available
      if (session.performance) {
        console.log(`  Load Time: ${session.performance.loadCompleteTime}ms`);
      }
      
      // Show memory usage if available
      if (session.latestMemory) {
        const memoryMB = (session.latestMemory.usedJSHeapSize / 1024 / 1024).toFixed(2);
        console.log(`  Memory: ${memoryMB}MB`);
      }
      
      console.log('');
    });
    
    // Get details for the first session
    if (sessions[0] && sessions[0].sessionId) {
      console.log(`\nFetching details for session: ${sessions[0].sessionId}`);
      const details = await getTelemetrySession(sessions[0].sessionId);
      
      if (details && details.events) {
        console.log(`\nLast 5 events:`);
        const lastEvents = details.events.slice(-5);
        lastEvents.forEach(event => {
          console.log(`  - ${event.type}: ${JSON.stringify(event.data).substring(0, 100)}...`);
        });
      }
    }
  } else {
    console.log('No telemetry sessions found.');
    console.log('\nTo generate telemetry data:');
    console.log('1. Start Brummer with proxy enabled: brum --proxy');
    console.log('2. Run the telemetry test: npm run telemetry-test');
    console.log('3. Configure your browser to use proxy localhost:8888');
    console.log('4. Visit http://localhost:3457/');
  }
  
  // Monitor real-time events
  console.log('\n\nMonitoring real-time telemetry events...');
  console.log('(Press Ctrl+C to exit)\n');
  monitorTelemetryEvents();
}

// Run the demo
main().catch(console.error);