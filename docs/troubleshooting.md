# Troubleshooting Common Issues

## Instance Discovery Problems
```bash
# Check instance discovery
ls ~/.brum/instances/          # Should show JSON files

# Manually clean up stale instances
rm ~/.brum/instances/*.json

# Check instance connectivity
instances_list                 # Shows connection states
```

## Port Conflicts
```bash
# Find available port
brum --port 0                  # Auto-assign available port

# Check what's using a port
lsof -i :7777                  # macOS/Linux
netstat -ano | findstr :7777   # Windows
```

## Proxy Issues
```bash
# Reset proxy configuration
brum --no-proxy               # Disable proxy temporarily

# Check proxy mappings
proxy_requests                # Show captured requests

# Force URL re-detection
# Restart process to trigger URL detection
```

## Health Monitoring Debug
```bash
# Instance health information
instances_list | jq '.[] | {id, state, retry_count, time_in_state}'

# Connection state history
instances_list | jq '.[] | .state_stats'
```