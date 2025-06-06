#!/usr/bin/env node

// Test script to output various ANSI color codes
console.log('\x1b[31mThis is red text\x1b[0m');
console.log('\x1b[32mThis is green text\x1b[0m');
console.log('\x1b[33mThis is yellow text\x1b[0m');
console.log('\x1b[34mThis is blue text\x1b[0m');
console.log('\x1b[35mThis is magenta text\x1b[0m');
console.log('\x1b[36mThis is cyan text\x1b[0m');
console.log('\x1b[37mThis is white text\x1b[0m');

// Bold text
console.log('\x1b[1m\x1b[31mThis is bold red text\x1b[0m');

// Background colors
console.log('\x1b[41mRed background\x1b[0m');
console.log('\x1b[42mGreen background\x1b[0m');

// Underlined text
console.log('\x1b[4mThis is underlined text\x1b[0m');

// Combined styles
console.log('\x1b[1m\x1b[33m\x1b[44mBold yellow text on blue background\x1b[0m');

// Simulating a build output with colors
setTimeout(() => {
    console.log('\x1b[32m✓\x1b[0m Building project...');
}, 1000);

setTimeout(() => {
    console.log('\x1b[33m⚠\x1b[0m Warning: deprecated API usage');
}, 2000);

setTimeout(() => {
    console.log('\x1b[31m✗\x1b[0m Error: compilation failed');
}, 3000);

setTimeout(() => {
    console.log('\x1b[36mℹ\x1b[0m Info: Process completed');
    process.exit(0);
}, 4000);