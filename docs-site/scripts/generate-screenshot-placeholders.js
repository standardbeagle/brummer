#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// List of screenshots referenced in documentation
const screenshots = [
  // Getting Started
  'brummer-tui.png',
  
  // Tutorial screenshots
  'tutorial-first-launch.png',
  'tutorial-server-started.png',
  'tutorial-processes-view.png',
  'tutorial-multiple-processes.png',
  'tutorial-logs-view.png',
  'tutorial-errors-view.png',
  'tutorial-urls-view.png',
  
  // React examples
  'react-scripts.png',
  'react-dev-server.png',
  'react-tests.png',
  'react-multiple-processes.png',
  'react-typescript-error.png',
  'react-test-failure.png',
  'react-build-perf.png',
  
  // Next.js examples
  'nextjs-scripts.png',
  'nextjs-database.png',
  'nextjs-services.png',
  'nextjs-websocket.png',
  'nextjs-webhooks.png',
  'nextjs-build-error.png',
  'nextjs-bundle.png',
  'nextjs-prisma.png',
  'nextjs-e2e.png',
  
  // Monorepo examples
  'monorepo-overview.png',
  'monorepo-turbo.png',
  'monorepo-all-apps.png',
  'monorepo-tests.png',
  'monorepo-hmr.png',
  'monorepo-graph.png',
  'monorepo-parallel.png',
  'monorepo-ci.png',
  
  // Microservices examples
  'microservices-scripts.png',
  'microservices-infra.png',
  'microservices-starting.png',
  'microservices-health.png',
  'microservices-errors.png',
  'microservices-rabbitmq.png',
  'microservices-databases.png',
  'microservices-integration-tests.png',
  'microservices-metrics.png'
];

// SVG template for placeholder
const createPlaceholderSVG = (filename, width = 800, height = 600) => {
  return `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">
  <rect width="100%" height="100%" fill="#1a1a1a"/>
  <rect x="10" y="10" width="${width-20}" height="${height-20}" fill="#2d2d2d" stroke="#444" stroke-width="2" rx="8"/>
  
  <!-- Terminal header -->
  <rect x="10" y="10" width="${width-20}" height="40" fill="#333" rx="8"/>
  <circle cx="30" cy="30" r="6" fill="#ff5f56"/>
  <circle cx="50" cy="30" r="6" fill="#ffbd2e"/>
  <circle cx="70" cy="30" r="6" fill="#27c93f"/>
  <text x="${width/2}" y="35" text-anchor="middle" fill="#ccc" font-family="monospace" font-size="14">Brummer TUI</text>
  
  <!-- Content area -->
  <text x="${width/2}" y="${height/2}" text-anchor="middle" fill="#666" font-family="Arial" font-size="24">
    Screenshot Placeholder
  </text>
  <text x="${width/2}" y="${height/2 + 30}" text-anchor="middle" fill="#888" font-family="Arial" font-size="16">
    ${filename}
  </text>
  
  <!-- Footer hint -->
  <text x="${width/2}" y="${height - 30}" text-anchor="middle" fill="#555" font-family="monospace" font-size="12">
    Press ? for help | Tab to switch views | q to quit
  </text>
</svg>`;
};

// Create screenshots directory
const screenshotsDir = path.join(__dirname, '../static/img/screenshots');
if (!fs.existsSync(screenshotsDir)) {
  fs.mkdirSync(screenshotsDir, { recursive: true });
}

// Generate placeholder images
screenshots.forEach(filename => {
  const svgContent = createPlaceholderSVG(filename);
  const filePath = path.join(screenshotsDir, filename);
  
  // For now, save as SVG (can be converted to PNG later)
  const svgPath = filePath.replace('.png', '.svg');
  fs.writeFileSync(svgPath, svgContent);
  console.log(`Created placeholder: ${filename}`);
});

// Also create a simple PNG version using canvas (if available)
try {
  const { createCanvas } = require('canvas');
  
  screenshots.forEach(filename => {
    const canvas = createCanvas(800, 600);
    const ctx = canvas.getContext('2d');
    
    // Dark background
    ctx.fillStyle = '#1a1a1a';
    ctx.fillRect(0, 0, 800, 600);
    
    // Terminal window
    ctx.fillStyle = '#2d2d2d';
    ctx.strokeStyle = '#444';
    ctx.lineWidth = 2;
    roundRect(ctx, 10, 10, 780, 580, 8);
    
    // Terminal header
    ctx.fillStyle = '#333';
    roundRect(ctx, 10, 10, 780, 40, 8);
    
    // Window controls
    ctx.fillStyle = '#ff5f56';
    ctx.beginPath();
    ctx.arc(30, 30, 6, 0, Math.PI * 2);
    ctx.fill();
    
    ctx.fillStyle = '#ffbd2e';
    ctx.beginPath();
    ctx.arc(50, 30, 6, 0, Math.PI * 2);
    ctx.fill();
    
    ctx.fillStyle = '#27c93f';
    ctx.beginPath();
    ctx.arc(70, 30, 6, 0, Math.PI * 2);
    ctx.fill();
    
    // Text
    ctx.fillStyle = '#ccc';
    ctx.font = '14px monospace';
    ctx.textAlign = 'center';
    ctx.fillText('Brummer TUI', 400, 35);
    
    ctx.fillStyle = '#666';
    ctx.font = '24px Arial';
    ctx.fillText('Screenshot Placeholder', 400, 300);
    
    ctx.fillStyle = '#888';
    ctx.font = '16px Arial';
    ctx.fillText(filename, 400, 330);
    
    ctx.fillStyle = '#555';
    ctx.font = '12px monospace';
    ctx.fillText('Press ? for help | Tab to switch views | q to quit', 400, 570);
    
    // Save as PNG
    const buffer = canvas.toBuffer('image/png');
    fs.writeFileSync(path.join(screenshotsDir, filename), buffer);
    console.log(`Created PNG placeholder: ${filename}`);
  });
  
  function roundRect(ctx, x, y, width, height, radius) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.arc(x + width - radius, y + radius, radius, -Math.PI / 2, 0);
    ctx.lineTo(x + width, y + height - radius);
    ctx.arc(x + width - radius, y + height - radius, radius, 0, Math.PI / 2);
    ctx.lineTo(x + radius, y + height);
    ctx.arc(x + radius, y + height - radius, radius, Math.PI / 2, Math.PI);
    ctx.lineTo(x, y + radius);
    ctx.arc(x + radius, y + radius, radius, Math.PI, -Math.PI / 2);
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
  }
} catch (e) {
  console.log('\nNote: Install "canvas" package for PNG placeholders:');
  console.log('  npm install canvas');
  console.log('\nSVG placeholders have been created instead.');
}

console.log(`\n✅ Created ${screenshots.length} placeholder screenshots`);
console.log('\nNext steps:');
console.log('1. Run Brummer and take actual screenshots');
console.log('2. Replace placeholders with real screenshots');
console.log('3. Optimize images for web (use tinypng.com or similar)');