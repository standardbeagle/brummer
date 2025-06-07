// Background script for the Brummer Firefox extension
console.log('Brummer DevTools extension loaded');

let brummerConnection = null;
let brummerClientId = null;
let brummerEndpoints = {};
let logQueue = [];

// Handle messages from content scripts or devtools
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === 'brummer_url_detected') {
        console.log('URL detected:', message.url);
        // Could potentially forward this to Brummer or store for later
    } else if (message.type === 'brummer_browser_log') {
        // Forward browser log to Brummer
        forwardLogToBrummer(message.data, sender.tab);
    } else if (message.type === 'brummer_connection_status') {
        // Update connection info from panel
        brummerConnection = message.connection;
        brummerClientId = message.clientId;
        
        // Process any queued logs
        if (brummerConnection && logQueue.length > 0) {
            logQueue.forEach(log => sendLogToBrummer(log));
            logQueue = [];
        }
    }
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