// Brummer DevTools Extension
console.log("🐝 Brummer DevTools extension loading...");

// Create a panel in the DevTools
chrome.devtools.panels.create(
    "🐝 Brummer",
    "icons/bee-32.png",
    "panel.html",
    function(panel) {
        console.log("✅ Brummer DevTools panel created successfully");
        
        // Store panel reference
        let panelWindow = null;
        
        // Panel lifecycle
        panel.onShown.addListener(function(window) {
            console.log("👁️ Brummer panel shown");
            panelWindow = window;
            
            // Initialize the panel if the function exists
            if (panelWindow.initializeBrummer) {
                panelWindow.initializeBrummer();
            }
            
            // Log current tab info for debugging
            chrome.devtools.inspectedWindow.eval(
                "location.href",
                function(result, isException) {
                    if (!isException) {
                        console.log("📍 Inspecting:", result);
                    }
                }
            );
        });
        
        panel.onHidden.addListener(function() {
            console.log("👻 Brummer panel hidden");
        });
        
        // Listen for messages from content scripts
        chrome.runtime.onMessage.addListener(function(request, sender, sendResponse) {
            console.log("📨 Message received in devtools:", request);
            
            // Forward relevant messages to the panel
            if (panelWindow && request.type && request.type.startsWith('brummer_')) {
                panelWindow.postMessage(request, '*');
            }
        });
    }
);

// Log that we're ready
console.log("🐝 Brummer DevTools extension ready!");