#!/bin/bash

# Script to update version numbers across all package files

if [ -z "$1" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.2.0"
    exit 1
fi

VERSION=$1
VERSION_WITH_V="v$VERSION"

echo "Updating version to $VERSION..."

# Update package.json
sed -i.bak "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" package.json && rm package.json.bak

# Update Chocolatey nuspec
sed -i.bak "s/<version>.*<\/version>/<version>$VERSION<\/version>/" chocolatey/brummer.nuspec && rm chocolatey/brummer.nuspec.bak

# Update Chocolatey install script
sed -i.bak "s/download\/v[0-9.]*\//download\/$VERSION_WITH_V\//" chocolatey/tools/chocolateyInstall.ps1 && rm chocolatey/tools/chocolateyInstall.ps1.bak

# Update Winget manifests
find winget -name "*.yaml" -exec sed -i.bak "s/PackageVersion: .*/PackageVersion: $VERSION/" {} \; -exec rm {}.bak \;
find winget -name "*.yaml" -exec sed -i.bak "s/download\/v[0-9.]*\//download\/$VERSION_WITH_V\//" {} \; -exec rm {}.bak \;

# Update Homebrew formula
sed -i.bak "s/version \".*\"/version \"$VERSION\"/" homebrew/brummer.rb && rm homebrew/brummer.rb.bak

echo "✅ Version updated to $VERSION"
echo ""
echo "Next steps:"
echo "1. Update SHA256 checksums after building releases"
echo "2. Commit changes: git commit -am 'Bump version to $VERSION'"
echo "3. Create git tag: git tag $VERSION_WITH_V"
echo "4. Push changes: git push && git push --tags"