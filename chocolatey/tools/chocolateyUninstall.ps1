$ErrorActionPreference = 'Stop'

$packageName = 'brummer'

# Remove shims
Uninstall-BinFile -Name 'brum'
Uninstall-BinFile -Name 'brummer'