// JavaScript file with common runtime errors for React projects

// Console error scenarios
console.log('Starting error scenario tests...');

// 1. Reference Errors
try {
  console.log(undefinedVariable);
} catch (error) {
  console.error('ReferenceError caught:', error.message);
}

// 2. Type Errors
try {
  const nullVar = null;
  nullVar.someMethod();
} catch (error) {
  console.error('TypeError caught:', error.message);
}

// 3. Syntax Errors (this would prevent script from running)
// eval('function test() { console.log("missing closing brace"');

// 4. Network Errors
fetch('https://nonexistent-domain.invalid/api/data')
  .then(response => response.json())
  .catch(error => {
    console.error('Network Error:', error.message);
  });

// 5. JSON Parse Errors
try {
  JSON.parse('{invalid json}');
} catch (error) {
  console.error('JSON Parse Error:', error.message);
}

// 6. Promise Rejection
Promise.reject(new Error('Promise rejection test'))
  .catch(error => {
    console.error('Promise Rejection:', error.message);
  });

// 7. Async/Await Errors
async function asyncErrorTest() {
  try {
    const response = await fetch('https://invalid-api.nonexistent');
    const data = await response.json();
    return data;
  } catch (error) {
    console.error('Async Error:', error.message);
    throw error;
  }
}

asyncErrorTest().catch(error => {
  console.error('Unhandled async error:', error.message);
});

// 8. DOM Errors
try {
  document.getElementById('nonexistent').innerHTML = 'test';
} catch (error) {
  console.error('DOM Error:', error.message);
}

// 9. Local Storage Errors (in some environments)
try {
  localStorage.setItem('test', 'value');
} catch (error) {
  console.error('Storage Error:', error.message);
}

// 10. Module Import Errors (would show in webpack/build output)
// import('./nonexistent-module').catch(error => {
//   console.error('Dynamic Import Error:', error.message);
// });

console.log('Error scenario tests completed.');