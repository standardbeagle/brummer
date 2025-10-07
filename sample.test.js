// Simple Node.js test runner for demonstration
const { performance } = require('perf_hooks');

// Test results storage
const testResults = {
  total: 0,
  passed: 0,
  failed: 0,
  skipped: 0,
  tests: []
};

// Simple test assertion function
function assert(condition, message) {
  if (!condition) {
    throw new Error(message || 'Assertion failed');
  }
}

// Test runner function
function runTest(testName, testFn) {
  const startTime = performance.now();
  testResults.total++;

  try {
    console.log(`Running ${testName}...`);
    testFn();
    const duration = performance.now() - startTime;

    testResults.passed++;
    testResults.tests.push({
      name: testName,
      status: 'passed',
      duration: duration,
      message: null
    });

    console.log(`✓ ${testName} (${duration.toFixed(2)}ms)`);
  } catch (error) {
    const duration = performance.now() - startTime;

    testResults.failed++;
    testResults.tests.push({
      name: testName,
      status: 'failed',
      duration: duration,
      message: error.message
    });

    console.log(`✗ ${testName} - ${error.message} (${duration.toFixed(2)}ms)`);
  }
}

// Test functions
function testBasicAddition() {
  const result = 2 + 3;
  assert(result === 5, '2 + 3 should equal 5');
}

function testStringOperations() {
  const str = 'hello';
  assert(str.toUpperCase() === 'HELLO', 'String should uppercase correctly');
  assert(str.length === 5, 'String length should be 5');
}

function testArrayOperations() {
  const arr = [1, 2, 3];
  assert(arr.length === 3, 'Array length should be 3');
  assert(arr.includes(2), 'Array should include 2');

  // Simulate a slow operation
  const start = performance.now();
  while (performance.now() - start < 50) {
    // Busy wait to simulate work
  }
}

function testAsyncOperation() {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      resolve('async result');
    }, 10);
  });
}

function testFailingCase() {
  assert(false, 'This test should fail for demonstration');
}

function testSkippedCase() {
  console.log('⚠️  Test skipped for demonstration');
  testResults.skipped++;
  testResults.total--;
}

// Simulate a performance test
function testPerformance() {
  const start = performance.now();

  // Simulate some computational work
  let result = 0;
  for (let i = 0; i < 10000; i++) {
    result += Math.sqrt(i);
  }

  const duration = performance.now() - start;

  if (duration > 100) {
    console.log(`⚠️  Performance test took ${duration.toFixed(2)}ms (slow)`);
  } else {
    console.log(`✓ Performance test completed in ${duration.toFixed(2)}ms`);
  }
}

// Run tests
console.log('🧪 Running Node.js Tests...\n');

runTest('Basic Addition', testBasicAddition);
runTest('String Operations', testStringOperations);
runTest('Array Operations', testArrayOperations);
runTest('Performance Test', testPerformance);
runTest('Failing Case', testFailingCase);
runTest('Skipped Case', testSkippedCase);

// Run async test
runTest('Async Operation', async () => {
  const result = await testAsyncOperation();
  assert(result === 'async result', 'Async operation should return correct result');
});

// Print summary
console.log('\n📊 Test Results:');
console.log(`Total: ${testResults.total}`);
console.log(`Passed: ${testResults.passed}`);
console.log(`Failed: ${testResults.failed}`);
console.log(`Skipped: ${testResults.skipped}`);

if (testResults.failed > 0) {
  console.log('\n❌ Some tests failed!');
  process.exit(1);
} else {
  console.log('\n✅ All tests passed!');
  process.exit(0);
}