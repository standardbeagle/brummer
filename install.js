#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const https = require('https');
const { execSync } = require('child_process');

const packageJson = require('./package.json');

// Platform and architecture mapping
const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows'
};

const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64'
};

function getPlatformInfo() {
  const platform = platformMap[process.platform];
  const arch = archMap[process.arch];
  
  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }
  
  return { platform, arch };
}

function getBinaryName(platform, arch) {
  const extension = platform === 'windows' ? '.exe' : '';
  return `brum-${platform}-${arch}${extension}`;
}

function getDownloadUrl(version, binaryName) {
  return `https://github.com/standardbeagle/brummer/releases/download/v${version}/${binaryName}`;
}

function downloadFile(url, destination) {
  return new Promise((resolve, reject) => {
    console.log(`Downloading ${url}...`);
    
    const file = fs.createWriteStream(destination);
    const request = https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return downloadFile(response.headers.location, destination).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Download failed: ${response.statusCode} ${response.statusMessage}`));
        return;
      }
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        resolve();
      });
    });
    
    request.on('error', reject);
    file.on('error', reject);
  });
}

async function install() {
  try {
    console.log('Installing Brummer binary...');
    
    const { platform, arch } = getPlatformInfo();
    const version = packageJson.version;
    const binaryName = getBinaryName(platform, arch);
    
    // Create bin directory
    const binDir = path.join(__dirname, 'bin');
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    const executableName = platform === 'windows' ? 'brum.exe' : 'brum';
    const executablePath = path.join(binDir, executableName);
    
    // Try to download from GitHub releases first
    const downloadUrl = getDownloadUrl(version, binaryName);
    
    try {
      await downloadFile(downloadUrl, executablePath);
      console.log(`Downloaded binary to ${executablePath}`);
    } catch (downloadError) {
      console.log('GitHub release not found, checking for local dist files...');
      
      // Fallback to local dist files (for development/local builds)
      const localBinaryPath = path.join(__dirname, 'dist', binaryName);
      if (fs.existsSync(localBinaryPath)) {
        fs.copyFileSync(localBinaryPath, executablePath);
        console.log(`Copied local binary to ${executablePath}`);
      } else {
        throw new Error(`Neither remote nor local binary found for ${platform}-${arch}`);
      }
    }
    
    // Make executable
    if (platform !== 'windows') {
      fs.chmodSync(executablePath, '755');
    }
    
    console.log('✅ @standardbeagle/brum installed successfully!');
    console.log('You can now run: brum');
    
  } catch (error) {
    console.error('❌ Installation failed:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  install();
}

module.exports = { install };