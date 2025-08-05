// Test script that generates continuous logs
let counter = 0;

function logMessage() {
    counter++;
    console.log(`[${new Date().toISOString()}] Log message #${counter}`);
    
    // Generate some errors occasionally
    if (counter % 5 === 0) {
        console.error(`[ERROR] Something went wrong at message #${counter}`);
    }
    
    // Generate some warnings
    if (counter % 3 === 0) {
        console.warn(`[WARN] Warning at message #${counter}`);
    }
}

// Log every second
setInterval(logMessage, 1000);

console.log('Test script started - generating logs every second...');
console.log('Press Ctrl+C to stop');