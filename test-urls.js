#!/usr/bin/env node

// Test script to verify URL deduplication
console.log("Starting server at http://localhost:3000");
console.log("Another message mentioning http://localhost:3000");
console.log("Different URL: http://localhost:8080");
console.log("Yet another mention of http://localhost:3000");
console.log("API endpoint: http://localhost:3000/api/users");
console.log("Same API endpoint again: http://localhost:3000/api/users");

// Keep running for a bit then exit
setTimeout(() => {
  console.log("Server shutting down...");
  process.exit(0);
}, 5000);