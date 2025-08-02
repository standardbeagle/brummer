/***
{
  "description": "Example script showing how to create a custom debugging function for the REPL library",
  "category": "examples",
  "tags": ["example", "demo", "debugging"],
  "author": "Brummer User",
  "version": "1.0.0",
  "examples": [
    "highlightElement('button')",
    "highlightElement('#login-form', 'red')",
    "highlightElement('.navbar', 'blue', 3000)"
  ],
  "parameters": {
    "selector": "CSS selector for the element to highlight",
    "color": "Optional highlight color (default: 'yellow')",
    "duration": "Optional duration in milliseconds (default: 2000)"
  },
  "returnType": "Object with element info and highlight status"
}
***/

function highlightElement(selector, color = 'yellow', duration = 2000) {
  const element = document.querySelector(selector);
  if (!element) {
    return { error: 'Element not found: ' + selector };
  }

  // Store original style
  const originalOutline = element.style.outline;
  const originalBackground = element.style.backgroundColor;
  
  // Apply highlight
  element.style.outline = `3px solid ${color}`;
  element.style.backgroundColor = color + '33'; // Add transparency
  
  // Remove highlight after duration
  setTimeout(() => {
    element.style.outline = originalOutline;
    element.style.backgroundColor = originalBackground;
  }, duration);
  
  return {
    success: true,
    element: {
      tagName: element.tagName.toLowerCase(),
      id: element.id || null,
      classes: Array.from(element.classList)
    },
    highlightColor: color,
    duration: duration,
    message: `Highlighted ${selector} for ${duration}ms`
  };
}