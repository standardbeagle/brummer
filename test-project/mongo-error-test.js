// Test script to simulate MongoDB connection error

console.log('[12:52:32] dev: Starting development server...');

// Simulate some normal logs
console.log('[12:52:32] dev: Loading configuration...');
console.log('[12:52:32] dev: Connecting to database...');

// Simulate the MongoDB error with proper formatting
setTimeout(() => {
  // Simulate the error structure from the user's example
  const error = {
    errorLabelSet: new Set(),
    reason: 'TopologyDescription',
    code: undefined,
    cause: {
      errorLabelSet: new Set(['ResetPool']),
      beforeHandshake: false,
      cause: {
        errno: -3008,
        code: 'ENOTFOUND',
        syscall: 'getaddrinfo',
        hostname: 'mongodb.localhost'
      }
    }
  };

  // Output similar to what the user showed
  console.error('[12:52:32] dev:  ⨯ unhandledRejection: [MongoServerSelectionError: getaddrinfo ENOTFOUND mongodb.localhost] {');
  console.error('[12:52:32] dev:   errorLabelSet: Set(0) {},');
  console.error('[12:52:32] dev:   reason: [TopologyDescription],');
  console.error('[12:52:32] dev:   code: undefined,');
  console.error('[12:52:32] dev:   [cause]: [MongoNetworkError: getaddrinfo ENOTFOUND mongodb.localhost] {');
  console.error('[12:52:32] dev:     errorLabelSet: Set(1) { \'ResetPool\' },');
  console.error('[12:52:32] dev:     beforeHandshake: false,');
  console.error('[12:52:32] dev:     [cause]: [Error: getaddrinfo ENOTFOUND mongodb.localhost] {');
  console.error('[12:52:32] dev:       errno: -3008,');
  console.error('[12:52:32] dev:       code: \'ENOTFOUND\',');
  console.error('[12:52:32] dev:       syscall: \'getaddrinfo\',');
  console.error('[12:52:32] dev:       hostname: \'mongodb.localhost\'');
  console.error('[12:52:32] dev:     }');
  console.error('[12:52:32] dev:   }');
  console.error('[12:52:32] dev: }');
  
  // Also show individual line errors like in the user's example
  console.error('[12:52:32] dev: }');
  console.error('[12:52:32] dev:   }');
  console.error('[12:52:32] dev:     }');
  console.error('[12:52:32] dev:       hostname: \'mongodb.localhost\'');
  console.error('[12:52:32] dev:       syscall: \'getaddrinfo\',');
  console.error('[12:52:32] dev:       code: \'ENOTFOUND\',');
  
  console.log('[12:52:33] dev: Continuing with other operations...');
  
  // Exit after a delay
  setTimeout(() => {
    console.log('[12:52:34] dev: Server shutting down...');
    process.exit(1);
  }, 1000);
}, 500);

// Keep the process running
setInterval(() => {
  console.log(`[${new Date().toTimeString().slice(0,8)}] dev: Server is running...`);
}, 5000);