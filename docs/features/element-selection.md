# Element Selection Feature

## Overview

The Element Selection feature allows you to inspect and send detailed information about any DOM element on a proxied page to the AI Coder. This is useful for debugging layout issues, understanding component structure, or asking the AI for help with specific elements.

## How to Use

### 1. Select an Element

1. Open the Brummer debug panel on any proxied page
2. Click the **"🎯 Select Element"** button
3. An overlay will appear with a crosshair cursor
4. Hover over elements to see them highlighted with a blue border
5. Click on any element to select it
6. Press **ESC** to cancel selection

### 2. Send Element Details to AI Coder

1. After selecting an element, type your question or request in the textarea
2. Click the **"🎯 + Text"** button
3. Your message and the element details will be sent to the AI Coder

## Element Details Captured

When you select an element, the following information is extracted and sent to the AI Coder:

### Basic Information
- **Tag Name**: The HTML tag (e.g., `div`, `button`, `input`)
- **ID**: The element's ID attribute (if present)
- **Class Name**: The element's class attribute(s)
- **All Attributes**: Complete list of HTML attributes

### Layout & Position
- **Bounds**: Precise position and size (x, y, width, height, top, left)
- **Computed Styles**: Key CSS properties including:
  - Display, position
  - Width, height
  - Margin, padding
  - Background color, text color
  - Font size, font family
  - Border, z-index

### Content & Context
- **Text Content**: First 200 characters of text content
- **Inner HTML**: First 500 characters of HTML content
- **CSS Selector Path**: A CSS selector path to locate the element (up to 5 levels deep)

## Example Use Cases

### Debugging Layout Issues
```
Select the misaligned button and ask:
"Why is this button not centered? How can I fix the alignment?"
```

### Understanding Component Structure
```
Select a complex component and ask:
"Explain the structure of this component and suggest improvements."
```

### Styling Questions
```
Select an element and ask:
"What CSS properties are making this element look like this?"
```

### Accessibility Improvements
```
Select a form input and ask:
"How can I improve the accessibility of this input field?"
```

## UI Buttons

The AI Coder section in the debug panel now has four buttons:

1. **📤 Send Text** - Send text message only
2. **📸 + Text** - Send text with full page screenshot
3. **🎯 Select Element** - Enter element selection mode
4. **🎯 + Text** - Send text with selected element details

## Technical Details

### Element Inspection
- Uses `getBoundingClientRect()` for accurate positioning
- Uses `getComputedStyle()` for actual rendered styles
- Filters out Brummer's own elements (prefixed with `__brummer`)
- Captures element hierarchy up to 5 levels

### Data Format
The element data is sent as a JSON object with the following structure:

```json
{
  "message": "User's message",
  "url": "Current page URL",
  "timestamp": 1234567890,
  "element": {
    "tagName": "button",
    "id": "submit-btn",
    "className": "btn btn-primary",
    "attributes": {...},
    "styles": {...},
    "bounds": {...},
    "selectorPath": "body > div.container > button#submit-btn",
    "textContent": "Submit",
    "innerHTML": "Submit"
  },
  "context": {
    "pageTitle": "Test Page",
    "viewport": {
      "width": 1920,
      "height": 1080
    }
  }
}
```

## Keyboard Shortcuts

- **ESC** - Cancel element selection mode

## Visual Feedback

- **Semi-transparent overlay** - Indicates selection mode is active
- **Blue highlight border** - Shows which element will be selected
- **Status messages** - Provide feedback on selection and submission
- **Button state changes** - Buttons update to show current state

## Implementation Notes

- Element selection is non-blocking and can be cancelled at any time
- Selected element state persists until sent or a new element is selected
- All event listeners are properly cleaned up when exiting selection mode
- The overlay has a high z-index (999998) to ensure it's visible above page content
- The highlight has an even higher z-index (999999) to appear above the overlay

## Related Features

- **Screenshot Capture** - Use `📸 + Text` to send a full page screenshot instead of element details
- **Text Messages** - Use `📤 Send Text` for simple text messages without context
- **AI Coder Integration** - All messages are sent to the active AI Coder session

## Troubleshooting

### Selection overlay doesn't appear
- Make sure the Brummer debug panel is visible
- Check browser console for JavaScript errors
- Refresh the page and try again

### Can't select certain elements
- Some elements may be hidden behind others
- Try selecting the parent element instead
- Use browser DevTools as an alternative for hard-to-reach elements

### Element details not sent
- Verify WebSocket connection is active (check status indicator)
- Ensure AI Coder session is running
- Check that you've selected an element before clicking "🎯 + Text"
