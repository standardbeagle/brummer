// Background script for the Brummer Firefox extension
console.log('🐝 Brummer background script starting...');

let brummerConnection = null;
let brummerClientId = null;
let brummerEndpoints = {};
let logQueue = [];
let activeTabs = new Map(); // Track tabs with Brummer parameters

// Log extension version and environment
// Note: chrome.management might not be available in all contexts
if (chrome.management && chrome.management.getSelf) {
    chrome.management.getSelf((extensionInfo) => {
        console.log(`🐝 Brummer Extension v${extensionInfo.version} loaded`);
        console.log(`📍 Extension ID: ${extensionInfo.id}`);
    });
} else {
    console.log('🐝 Brummer Extension loaded (manifest v3 service worker)');
}

// Handle messages from content scripts or devtools
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    console.log(`📨 Message received:`, message.type, sender.tab ? `from tab ${sender.tab.id}` : 'from extension');
    
    if (message.type === 'brummer_url_detected') {
        console.log('🔗 URL detected:', message.url);
        // Could potentially forward this to Brummer or store for later
    } else if (message.type === 'brummer_browser_log') {
        // Forward browser log to Brummer
        console.log('📝 Forwarding browser log from tab:', sender.tab?.id);
        forwardLogToBrummer(message.data, sender.tab);
    } else if (message.type === 'brummer_connection_status') {
        // Update connection info from panel
        console.log(`🔌 Connection status update:`, message.connection ? 'Connected' : 'Disconnected');
        brummerConnection = message.connection;
        brummerClientId = message.clientId;
        
        // Process any queued logs
        if (brummerConnection && logQueue.length > 0) {
            console.log(`📤 Processing ${logQueue.length} queued logs...`);
            logQueue.forEach(log => sendLogToBrummer(log));
            logQueue = [];
        }
    } else if (message.type === 'brummer_tab_activated') {
        // Track tabs with Brummer parameters
        if (sender.tab) {
            activeTabs.set(sender.tab.id, {
                id: sender.tab.id,
                url: sender.tab.url,
                title: sender.tab.title,
                token: message.token,
                process: message.process,
                lastSeen: new Date()
            });
            console.log(`🌐 Tab ${sender.tab.id} activated with Brummer logging`);
        }
    }
    
    // Send response to keep the message channel open
    sendResponse({received: true});
    return true; // Keep channel open for async responses
});

async function forwardLogToBrummer(logData, tab) {
    if (!brummerConnection) {
        // Queue the log if not connected
        logQueue.push({...logData, tab: tab});
        return;
    }
    
    // Enhance log data with tab information
    const enhancedLog = {
        ...logData,
        tab: {
            id: tab.id,
            url: tab.url,
            title: tab.title
        }
    };
    
    await sendLogToBrummer(enhancedLog);
}

async function sendLogToBrummer(logData) {
    if (!brummerConnection) return;
    
    try {
        const response = await fetch(`${brummerConnection}/api/browser-log`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                clientId: brummerClientId,
                logData: logData
            })
        });
        
        if (!response.ok) {
            console.warn('Failed to send log to Brummer:', response.status);
        }
    } catch (error) {
        console.error('Error sending log to Brummer:', error);
        // Queue for retry if connection is re-established
        logQueue.push(logData);
    }
}

// Set up context menu for opening URLs in Brummer (optional)
chrome.contextMenus.create({
    id: 'brummer-open',
    title: 'Open in Brummer DevTools',
    contexts: ['link']
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
    if (info.menuItemId === 'brummer-open') {
        // Could send the URL to Brummer or open DevTools
        console.log('Context menu clicked for URL:', info.linkUrl);
    }
});

// Enable/disable browser logging based on connection status
chrome.storage.onChanged.addListener((changes, namespace) => {
    if (changes.brummerLoggingEnabled) {
        // Notify all tabs about logging status change
        chrome.tabs.query({}, (tabs) => {
            tabs.forEach(tab => {
                chrome.tabs.sendMessage(tab.id, {
                    type: 'brummer_toggle_logging',
                    enabled: changes.brummerLoggingEnabled.newValue
                }).catch(() => {
                    // Ignore errors for tabs that don't have content script
                });
            });
        });
    }
});