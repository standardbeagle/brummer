// Create a panel in the DevTools
chrome.devtools.panels.create(
    "🐝 Brummer",
    "icons/bee-32.png",
    "panel.html",
    function(panel) {
        console.log("Brummer DevTools panel created");
        
        // Panel lifecycle
        panel.onShown.addListener(function(panelWindow) {
            console.log("Brummer panel shown");
            if (panelWindow.initializeBrummer) {
                panelWindow.initializeBrummer();
            }
        });
        
        panel.onHidden.addListener(function() {
            console.log("Brummer panel hidden");
        });
    }
);