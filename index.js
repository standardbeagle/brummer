#!/usr/bin/env node

// This file exists to allow programmatic usage if needed
// The main entry point is the binary in ./bin/brum

const { spawn } = require('child_process');
const path = require('path');

const binaryPath = path.join(__dirname, 'bin', process.platform === 'win32' ? 'brum.exe' : 'brum');

// Forward all arguments to the actual binary
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

child.on('error', (err) => {
  if (err.code === 'ENOENT') {
    console.error('Brummer binary not found. Please reinstall the package.');
    console.error('Run: npm install -g brummer');
    process.exit(1);
  }
  throw err;
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code);
  }
});