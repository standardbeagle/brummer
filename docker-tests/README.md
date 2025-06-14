# Brummer Package Deployment Tests

This directory contains Docker/Podman tests for validating Brummer package deployment across different platforms and package managers.

## Test Scenarios

### 📦 NPM Package Tests
- **npm-node18**: Test NPM package installation on Node.js 18 (Alpine)
- **npm-node20**: Test NPM package installation on Node.js 20 with Yarn support
- **ubuntu-npm**: Test NPM package installation on Ubuntu 22.04
- **alpine-pnpm**: Test PNPM package installation on Alpine Linux

### 🐹 Go Install Tests  
- **go-install**: Test Go module installation and building from source

### 🔄 Cross-Platform Tests
- **cross-platform**: Test binary platform detection and installation logic

## Running Tests

### Prerequisites
- Podman installed and configured
- Project built with `make build-all` (for binary tests)

### Run All Tests
```bash
./docker-tests/run-tests.sh
```

### Run Individual Tests
```bash
# Test NPM package on Node.js 18
podman build -f docker-tests/Dockerfile.npm-node18 -t brummer-test-npm-node18 .
podman run --rm brummer-test-npm-node18

# Test Go installation
podman build -f docker-tests/Dockerfile.go-install -t brummer-test-go-install .
podman run --rm brummer-test-go-install
```

## Test Coverage

Each test validates:
- ✅ Package manager installation
- ✅ Binary execution and help output
- ✅ Platform compatibility
- ✅ Installation script functionality
- ✅ Cross-platform binary selection

## Test Results

The tests verify:
1. **NPM Package (@standardbeagle/brum)**
   - Installs correctly across Node.js versions
   - `brum` command is available after installation
   - Works with npm, yarn, and pnpm
   - Platform detection works correctly

2. **Go Installation**
   - Builds from source successfully
   - Make targets work correctly
   - Binary installs to correct location

3. **Cross-Platform Compatibility**
   - Correct binary selection for platform/architecture
   - Installation script handles different environments
   - Binary permissions set correctly

## Debugging Failed Tests

If tests fail:
1. Check the container logs for specific error messages
2. Verify all binaries exist in `dist/` directory
3. Ensure `package.json` and `install.js` are correctly configured
4. Test individual components manually:
   ```bash
   node install.js
   ./bin/brum --help
   ```

## Adding New Tests

To add a new test scenario:
1. Create `Dockerfile.{test-name}` in this directory
2. Add test to the `TESTS` array in `run-tests.sh`
3. Follow the existing pattern for test scripts
4. Update this README with the new test description