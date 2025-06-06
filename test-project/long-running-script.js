console.log('Starting long-running script...');

let count = 0;
const interval = setInterval(() => {
    count++;
    console.log(`Script running... ${count}`);
    
    if (count % 5 === 0) {
        console.log(`✅ Checkpoint reached: ${count} iterations`);
    }
    
    if (count % 10 === 0) {
        console.log(`🌐 Example URL: http://localhost:${3000 + (count % 10)}/api/status`);
    }
}, 2000);

// Handle graceful shutdown
process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    clearInterval(interval);
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down...');
    clearInterval(interval);
    process.exit(0);
});

console.log('Long-running script initialized. Press Ctrl+C to stop.');