#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { promisify } = require('util');
const { pipeline } = require('stream');
const streamPipeline = promisify(pipeline);

// Configuration
const REPO = 'beagle/brummer';
const BINARY_NAME = 'brum';

// Detect platform
function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;
  
  let os;
  switch (platform) {
    case 'darwin':
      os = 'darwin';
      break;
    case 'linux':
      os = 'linux';
      break;
    case 'win32':
      os = 'windows';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }
  
  let architecture;
  switch (arch) {
    case 'x64':
      architecture = 'amd64';
      break;
    case 'arm64':
      architecture = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }
  
  return `${os}-${architecture}`;
}

// Get latest release version
async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/latest`,
      headers: {
        'User-Agent': 'brummer-npm-installer'
      }
    };
    
    https.get(options, (res) => {
      let data = '';
      
      res.on('data', (chunk) => {
        data += chunk;
      });
      
      res.on('end', () => {
        try {
          const release = JSON.parse(data);
          if (release.tag_name) {
            resolve(release.tag_name);
          } else {
            reject(new Error('Could not find latest release'));
          }
        } catch (e) {
          reject(e);
        }
      });
    }).on('error', reject);
  });
}

// Download file
async function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        file.close();
        fs.unlinkSync(dest);
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        file.close();
        fs.unlinkSync(dest);
        reject(new Error(`Failed to download: ${response.statusCode}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close(() => {
          fs.chmodSync(dest, 0o755);
          resolve();
        });
      });
      
      file.on('error', (err) => {
        fs.unlinkSync(dest);
        reject(err);
      });
    }).on('error', (err) => {
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

// Main installation
async function install() {
  try {
    console.log('🐝 Installing Brummer binary...');
    
    // Detect platform
    const platform = getPlatform();
    console.log(`Platform: ${platform}`);
    
    // Get latest version
    console.log('Fetching latest version...');
    const version = await getLatestVersion();
    console.log(`Version: ${version}`);
    
    // Construct download URL
    const binaryExt = process.platform === 'win32' ? '.exe' : '';
    const binaryName = `${BINARY_NAME}-${platform}${binaryExt}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${binaryName}`;
    
    // Download binary
    const binDir = path.join(__dirname, '..', 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    const localBinaryName = `${BINARY_NAME}${binaryExt}`;
    const binaryPath = path.join(binDir, localBinaryName);
    
    console.log(`Downloading from ${downloadUrl}...`);
    await downloadFile(downloadUrl, binaryPath);
    
    console.log('✅ Brummer installed successfully!');
    console.log('Run "brum" to get started.');
  } catch (error) {
    console.error('❌ Installation failed:', error.message);
    console.error('\nYou can try installing from source instead:');
    console.error('  git clone https://github.com/beagle/brummer');
    console.error('  cd brummer');
    console.error('  make install-user');
    process.exit(1);
  }
}

// Skip in CI or if explicitly disabled
if (process.env.CI || process.env.SKIP_BRUMMER_DOWNLOAD) {
  console.log('Skipping binary download');
  process.exit(0);
}

// Run installation
install();