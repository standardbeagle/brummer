class BrummerConnection {
    constructor() {
        this.serverUrl = 'http://localhost:7777';
        this.connected = false;
        this.eventSource = null;
        this.urls = [];
        this.clientId = null;
        
        this.loadSettings();
        this.updateStatus();
    }
    
    async loadSettings() {
        try {
            const result = await new Promise(resolve => {
                chrome.storage.local.get(['brummerServerUrl', 'brummerLoggingEnabled'], resolve);
            });
            
            if (result.brummerServerUrl) {
                this.serverUrl = result.brummerServerUrl;
                document.getElementById('serverUrl').value = this.serverUrl;
            }
            
            // Load browser logging setting
            const loggingEnabled = result.brummerLoggingEnabled || false;
            document.getElementById('browserLoggingToggle').checked = loggingEnabled;
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    }
    
    async saveSettings() {
        try {
            await new Promise(resolve => {
                chrome.storage.local.set({
                    brummerServerUrl: this.serverUrl
                }, resolve);
            });
        } catch (error) {
            console.error('Failed to save settings:', error);
        }
    }
    
    async connect() {
        try {
            this.serverUrl = document.getElementById('serverUrl').value.trim();
            await this.saveSettings();
            
            this.showError('');
            
            // First, connect to the MCP server
            const connectResponse = await fetch(`${this.serverUrl}/mcp/connect`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    clientName: 'Firefox DevTools'
                })
            });
            
            if (!connectResponse.ok) {
                throw new Error(`Failed to connect: ${connectResponse.status}`);
            }
            
            const connectData = await connectResponse.json();
            this.clientId = connectData.clientId;
            
            // Load initial URLs
            await this.loadUrls();
            
            // Set up real-time updates
            this.setupEventSource();
            
            this.connected = true;
            this.updateStatus();
            
            // Notify background script of connection status
            chrome.runtime.sendMessage({
                type: 'brummer_connection_status',
                connection: this.serverUrl,
                clientId: this.clientId
            });
            
        } catch (error) {
            console.error('Connection failed:', error);
            this.showError(`Failed to connect to Brummer: ${error.message}`);
            this.connected = false;
            this.updateStatus();
            
            // Notify background script of disconnection
            chrome.runtime.sendMessage({
                type: 'brummer_connection_status',
                connection: null,
                clientId: null
            });
        }
    }
    
    disconnect() {
        if (this.eventSource) {
            this.eventSource.close();
            this.eventSource = null;
        }
        
        this.connected = false;
        this.clientId = null;
        this.updateStatus();
        this.showError('');
        
        // Notify background script of disconnection
        chrome.runtime.sendMessage({
            type: 'brummer_connection_status',
            connection: null,
            clientId: null
        });
        
        const container = document.getElementById('urlContainer');
        container.innerHTML = '<div class="loading">Disconnected</div>';
    }
    
    setupEventSource() {
        if (this.eventSource) {
            this.eventSource.close();
        }
        
        this.eventSource = new EventSource(`${this.serverUrl}/mcp/events?clientId=${this.clientId}`);
        
        this.eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'log.line') {
                    // Refresh URLs when new logs come in
                    this.loadUrls();
                }
            } catch (error) {
                console.error('Failed to parse event:', error);
            }
        };
        
        this.eventSource.onerror = (error) => {
            console.error('EventSource error:', error);
            this.connected = false;
            this.updateStatus();
        };
    }
    
    async loadUrls() {
        try {
            const response = await fetch(`${this.serverUrl}/mcp/logs?processId=`);
            if (!response.ok) {
                throw new Error(`Failed to fetch logs: ${response.status}`);
            }
            
            const logs = await response.json();
            
            // Extract URLs from logs
            const urlPattern = /https?:\/\/[^\s<>"{}|\\^`\[\]]+/g;
            const foundUrls = [];
            
            logs.forEach(log => {
                const matches = log.content.match(urlPattern);
                if (matches) {
                    matches.forEach(url => {
                        // Skip if we already have this URL
                        if (!foundUrls.some(existing => existing.url === url)) {
                            foundUrls.push({
                                url: url,
                                timestamp: log.timestamp,
                                processName: log.processName,
                                context: log.content
                            });
                        }
                    });
                }
            });
            
            // Sort by timestamp (most recent first)
            foundUrls.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
            
            this.urls = foundUrls;
            this.renderUrls();
            
        } catch (error) {
            console.error('Failed to load URLs:', error);
            this.showError(`Failed to load URLs: ${error.message}`);
        }
    }
    
    renderUrls() {
        const container = document.getElementById('urlContainer');
        
        if (this.urls.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="bee-large">🐝</div>
                    <div>No URLs detected yet</div>
                    <div style="font-size: 12px; margin-top: 8px; color: #999;">
                        Start some scripts in Brummer to see detected URLs here
                    </div>
                </div>
            `;
            return;
        }
        
        const urlsHtml = this.urls.map(urlData => {
            const time = new Date(urlData.timestamp).toLocaleTimeString();
            const truncatedContext = urlData.context.length > 100 
                ? urlData.context.substring(0, 100) + '...'
                : urlData.context;
                
            return `
                <div class="url-item">
                    <div style="flex: 1;">
                        <div>
                            <a href="${urlData.url}" class="url-link" target="_blank">${urlData.url}</a>
                            <button class="open-btn" onclick="brummer.openUrl('${urlData.url}')">Open</button>
                        </div>
                        <div class="url-context">${truncatedContext}</div>
                    </div>
                    <div class="url-time">${time}</div>
                </div>
            `;
        }).join('');
        
        container.innerHTML = `<div class="url-list">${urlsHtml}</div>`;
    }
    
    openUrl(url) {
        // Open URL in a new tab
        chrome.tabs.create({ url: url });
    }
    
    updateStatus() {
        const statusEl = document.getElementById('status');
        if (this.connected) {
            statusEl.textContent = 'Connected';
            statusEl.className = 'status connected';
        } else {
            statusEl.textContent = 'Disconnected';
            statusEl.className = 'status disconnected';
        }
    }
    
    showError(message) {
        const container = document.getElementById('errorContainer');
        if (message) {
            container.innerHTML = `<div class="error-message">${message}</div>`;
        } else {
            container.innerHTML = '';
        }
    }
}

// Global instance
let brummer;

// Initialize when the panel is shown
function initializeBrummer() {
    if (!brummer) {
        brummer = new BrummerConnection();
        // Auto-connect on first load
        setTimeout(() => brummer.connect(), 500);
    }
}

// Global functions for HTML onclick handlers
function connect() {
    brummer.connect();
}

function disconnect() {
    brummer.disconnect();
}

async function toggleBrowserLogging() {
    const enabled = document.getElementById('browserLoggingToggle').checked;
    
    try {
        await new Promise(resolve => {
            chrome.storage.local.set({
                brummerLoggingEnabled: enabled
            }, resolve);
        });
        
        console.log('Browser logging', enabled ? 'enabled' : 'disabled');
    } catch (error) {
        console.error('Failed to save browser logging setting:', error);
        // Revert toggle on error
        document.getElementById('browserLoggingToggle').checked = !enabled;
    }
}

// Initialize immediately if panel is already visible
document.addEventListener('DOMContentLoaded', initializeBrummer);