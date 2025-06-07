// Test script with multiple types of errors

console.log('Starting multi-error test...');

// Test 1: ReferenceError
setTimeout(() => {
  try {
    console.log(undefinedVariable);
  } catch (e) {
    console.error(e);
  }
}, 100);

// Test 2: TypeError
setTimeout(() => {
  try {
    const obj = null;
    obj.someMethod();
  } catch (e) {
    console.error('TypeError occurred:', e.message);
    console.error(e.stack);
  }
}, 200);

// Test 3: Custom Error with stack trace
setTimeout(() => {
  function deepFunction() {
    throw new Error('Something went wrong in deepFunction!');
  }
  
  function middleFunction() {
    deepFunction();
  }
  
  function topFunction() {
    try {
      middleFunction();
    } catch (e) {
      console.error('⨯ Custom Error:', e.message);
      console.error('Stack trace:');
      console.error(e.stack);
    }
  }
  
  topFunction();
}, 300);

// Test 4: Async rejection
setTimeout(() => {
  Promise.reject(new Error('Async operation failed'))
    .catch(e => {
      console.error('⨯ unhandledRejection: [PromiseError: ' + e.message + ']');
      console.error('  at async operation (test-file.js:45:10)');
      console.error('  at Promise.reject');
    });
}, 400);

// Test 5: Warning
setTimeout(() => {
  console.warn('⚠️ Warning: Deprecated function used');
  console.warn('  Please update to the new API');
}, 500);

// Keep running
setInterval(() => {
  console.log('Application is still running...');
}, 5000);