# Windows Code Signing for Package Managers

This guide covers how to sign Windows binaries for Winget and Chocolatey package distribution.

## Overview

Windows package managers require signed binaries for security and trust. Here's what you need:

- **Code Signing Certificate** (EV or Standard)
- **Signing Tools** (SignTool, osslsigncode)
- **Timestamping Service** (for long-term validity)

## Certificate Options

### 1. Extended Validation (EV) Code Signing Certificate ⭐ **Recommended**
- **Cost**: $300-500/year
- **Benefits**: 
  - Immediate trust (no SmartScreen warnings)
  - Required for kernel-mode drivers
  - Better reputation with Windows Defender
- **Providers**: DigiCert, Sectigo, GlobalSign
- **Hardware**: Requires USB token or HSM

### 2. Standard Code Signing Certificate
- **Cost**: $100-200/year  
- **Benefits**: Basic signing capabilities
- **Drawbacks**: Initial SmartScreen warnings until reputation is built
- **Providers**: DigiCert, Sectigo, Comodo, GoDaddy

### 3. Free Options (Limited Use)
- **SignPath.io**: Free for open source projects
- **GitHub Actions**: Free CI/CD signing for OSS projects

## Getting a Certificate

### DigiCert (Recommended)
```bash
# 1. Create account at DigiCert
# 2. Verify organization identity
# 3. Purchase EV Code Signing Certificate
# 4. Complete validation process (phone calls, documents)
# 5. Receive USB token with certificate
```

### SignPath.io (Free for OSS)
```bash
# 1. Sign up at signpath.io
# 2. Submit open source project for approval
# 3. Set up CI/CD integration
# 4. Automatic signing on releases
```

## Signing Process

### Method 1: Windows SignTool (Windows only)
```powershell
# Install Windows SDK or Visual Studio
# Certificate must be in Windows Certificate Store

# Sign the binary
signtool sign /f "certificate.pfx" /p "password" /t http://timestamp.digicert.com /v brum-windows-amd64.exe

# Verify signature
signtool verify /pa brum-windows-amd64.exe
```

### Method 2: osslsigncode (Cross-platform)
```bash
# Install osslsigncode
# Ubuntu/Debian: apt install osslsigncode
# macOS: brew install osslsigncode

# Sign the binary
osslsigncode sign \
  -certs certificate.crt \
  -key private.key \
  -t http://timestamp.digicert.com \
  -in brum-windows-amd64.exe \
  -out brum-windows-amd64-signed.exe

# Verify signature
osslsigncode verify brum-windows-amd64-signed.exe
```

### Method 3: GitHub Actions (Automated)
```yaml
# .github/workflows/sign-windows.yml
name: Sign Windows Binaries

on:
  release:
    types: [published]

jobs:
  sign:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Download Binaries
        run: |
          # Download release binaries
          
      - name: Sign Binaries
        run: |
          # Import certificate from secrets
          echo "${{ secrets.SIGNING_CERT }}" | base64 -d > cert.pfx
          
          # Sign each Windows binary
          signtool sign /f cert.pfx /p "${{ secrets.CERT_PASSWORD }}" /t http://timestamp.digicert.com *.exe
          
      - name: Upload Signed Binaries
        # Upload back to release
```

## Timestamping Services

Always use timestamping to ensure signatures remain valid after certificate expires:

```bash
# DigiCert (Recommended)
http://timestamp.digicert.com

# Sectigo
http://timestamp.sectigo.com

# GlobalSign  
http://timestamp.globalsign.com
```

## Package Manager Requirements

### Winget
- **Signature**: Required for submission
- **Certificate**: EV preferred, Standard accepted
- **Validation**: Microsoft validates during submission
- **SmartScreen**: EV certificates bypass warnings

### Chocolatey
- **Signature**: Strongly recommended
- **Certificate**: Any valid code signing certificate
- **Moderation**: Signed packages get faster approval
- **Trust**: Reduces user warnings

## Signing Integration with Build Process

### Update Makefile
```makefile
# Add signing targets
.PHONY: sign-windows
sign-windows:
	@echo "🖊️ Signing Windows binaries..."
	@if [ -f "$(CERT_FILE)" ]; then \
		osslsigncode sign -certs $(CERT_FILE) -key $(KEY_FILE) \
			-t http://timestamp.digicert.com \
			-in dist/brum-windows-amd64.exe \
			-out dist/brum-windows-amd64-signed.exe && \
		osslsigncode sign -certs $(CERT_FILE) -key $(KEY_FILE) \
			-t http://timestamp.digicert.com \
			-in dist/brum-windows-arm64.exe \
			-out dist/brum-windows-arm64-signed.exe && \
		echo "✅ Windows binaries signed"; \
	else \
		echo "⚠️ Certificate not found, skipping signing"; \
	fi

# Update build-all to include signing
build-all: build-binaries sign-windows
```

### Environment Variables
```bash
# Set these in your environment or CI/CD
export CERT_FILE="path/to/certificate.crt"
export KEY_FILE="path/to/private.key"
export CERT_PASSWORD="your_certificate_password"
```

## Cost Breakdown

### Annual Costs
- **EV Certificate**: $300-500/year
- **Standard Certificate**: $100-200/year
- **HSM/Token**: Usually included with EV cert

### One-time Setup
- **Organization Validation**: Free (but requires time)
- **Document Preparation**: Free
- **Phone Verification**: Free

### Free Alternatives
- **SignPath.io**: Free for OSS projects
- **Self-signed**: Free (but not trusted)

## Security Best Practices

### Certificate Storage
```bash
# Store certificates securely
# Use environment variables for passwords
# Never commit certificates to git
# Use HSM or hardware tokens when possible
```

### Build Pipeline Security
```yaml
# Use GitHub secrets for certificates
# Limit access to signing workflows
# Use least-privilege principles
# Audit signing activities
```

### Validation
```bash
# Always verify signatures after signing
signtool verify /pa signed-binary.exe

# Check certificate details
signtool verify /v /pa signed-binary.exe
```

## Troubleshooting

### Common Issues

1. **"Certificate not trusted"**
   - Ensure certificate chain is complete
   - Install intermediate certificates
   - Use proper timestamping

2. **"Invalid signature"**
   - Check certificate expiration
   - Verify private key matches certificate
   - Ensure binary wasn't modified after signing

3. **SmartScreen warnings**
   - Use EV certificate for immediate trust
   - Build reputation over time with standard cert
   - Ensure consistent signing across releases

### Validation Commands
```bash
# Windows
signtool verify /pa /v binary.exe

# Cross-platform
osslsigncode verify binary.exe

# PowerShell
Get-AuthenticodeSignature binary.exe
```

## Next Steps

1. **Choose Certificate Type**: EV recommended for better trust
2. **Select Provider**: DigiCert, Sectigo, or SignPath.io for OSS
3. **Set Up Signing Pipeline**: Automate with CI/CD
4. **Test Thoroughly**: Verify on clean Windows systems
5. **Submit to Package Managers**: Follow their specific requirements

## Package Manager Submission

After signing, you'll need:
- **Winget**: Manifest files and PR to winget-pkgs repository
- **Chocolatey**: Nuspec files and package upload

See the respective package manager documentation for detailed submission processes.