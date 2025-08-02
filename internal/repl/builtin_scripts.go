package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BuiltinScript represents a script that comes built-in with Brummer
type BuiltinScript struct {
	Name     string
	Metadata ScriptMetadata
	Code     string
}

var (
	// builtinScriptsOnce ensures builtin scripts are loaded only once
	builtinScriptsOnce sync.Once
	// builtinScripts stores the loaded builtin scripts
	builtinScripts []BuiltinScript
)

// GetBuiltinScripts returns all built-in debugging scripts with lazy loading
func GetBuiltinScripts() []BuiltinScript {
	builtinScriptsOnce.Do(func() {
		builtinScripts = loadBuiltinScripts()
	})
	return builtinScripts
}

// loadBuiltinScripts loads the actual builtin scripts
func loadBuiltinScripts() []BuiltinScript {
	return []BuiltinScript{
		{
			Name: "getDetails",
			Metadata: ScriptMetadata{
				Description: "Get comprehensive details about a DOM element including CSS rules, styles, event listeners, parents, and layout contexts",
				Category:    "debugging",
				Tags:        []string{"dom", "css", "layout", "debugging"},
				Author:      "Brummer",
				Version:     "1.0.0",
				Examples: []string{
					"getDetails('button')",
					"getDetails('#login-form')",
					"getDetails('.navbar')",
				},
				Parameters: map[string]string{
					"selector": "CSS selector string to identify the element",
				},
				ReturnType: "Object with element details, styles, layout info, and parent chain",
			},
			Code: `function getDetails(selector) {
  const element = document.querySelector(selector);
  if (!element) {
    return { error: 'Element not found: ' + selector };
  }

  const computedStyle = window.getComputedStyle(element);
  const rect = element.getBoundingClientRect();
  
  // Get event listeners (if possible)
  const getEventListeners = (el) => {
    try {
      // Try to use Chrome DevTools API if available
      if (window.getEventListeners) {
        return window.getEventListeners(el);
      } 
      // Fallback: look for jQuery events
      if (window.jQuery && window.jQuery._data) {
        const data = window.jQuery._data(el);
        return data && data.events ? data.events : {};
      }
      return 'Event listeners not accessible (try Chrome DevTools)';
    } catch (e) {
      return 'Event listeners not accessible: ' + e.message;
    }
  };

  // Check if element starts a new stacking context
  const startsStackingContext = () => {
    const style = computedStyle;
    return style.position !== 'static' && 
           (style.zIndex !== 'auto' || 
            style.opacity !== '1' || 
            style.transform !== 'none' ||
            style.filter !== 'none' ||
            style.mixBlendMode !== 'normal' ||
            style.isolation === 'isolate');
  };

  // Check if element is scrollable
  const isScrollable = () => {
    const style = computedStyle;
    return style.overflow === 'scroll' || 
           style.overflow === 'auto' || 
           style.overflowX === 'scroll' || 
           style.overflowX === 'auto' ||
           style.overflowY === 'scroll' || 
           style.overflowY === 'auto';
  };

  // Get parent chain with layout info
  const getParentChain = () => {
    const parents = [];
    let current = element.parentElement;
    
    while (current && current !== document.body) {
      const parentStyle = window.getComputedStyle(current);
      const parentRect = current.getBoundingClientRect();
      
      parents.push({
        tagName: current.tagName.toLowerCase(),
        id: current.id || null,
        classes: Array.from(current.classList),
        position: parentStyle.position,
        display: parentStyle.display,
        zIndex: parentStyle.zIndex,
        overflow: parentStyle.overflow,
        bounds: {
          x: parentRect.x,
          y: parentRect.y,
          width: parentRect.width,
          height: parentRect.height
        },
        startsStackingContext: parentStyle.position !== 'static' && parentStyle.zIndex !== 'auto',
        isScrollable: parentStyle.overflow === 'scroll' || parentStyle.overflow === 'auto'
      });
      
      current = current.parentElement;
    }
    
    return parents;
  };

  return {
    element: {
      tagName: element.tagName.toLowerCase(),
      id: element.id || null,
      classes: Array.from(element.classList),
      attributes: Array.from(element.attributes).reduce((acc, attr) => {
        acc[attr.name] = attr.value;
        return acc;
      }, {})
    },
    bounds: {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
      top: rect.top,
      right: rect.right,
      bottom: rect.bottom,
      left: rect.left
    },
    computedStyles: {
      position: computedStyle.position,
      display: computedStyle.display,
      visibility: computedStyle.visibility,
      opacity: computedStyle.opacity,
      zIndex: computedStyle.zIndex,
      transform: computedStyle.transform,
      overflow: computedStyle.overflow,
      boxSizing: computedStyle.boxSizing,
      margin: {
        top: computedStyle.marginTop,
        right: computedStyle.marginRight,
        bottom: computedStyle.marginBottom,
        left: computedStyle.marginLeft
      },
      padding: {
        top: computedStyle.paddingTop,
        right: computedStyle.paddingRight,
        bottom: computedStyle.paddingBottom,
        left: computedStyle.paddingLeft
      },
      border: {
        top: computedStyle.borderTopWidth,
        right: computedStyle.borderRightWidth,
        bottom: computedStyle.borderBottomWidth,
        left: computedStyle.borderLeftWidth
      }
    },
    layoutContext: {
      startsStackingContext: startsStackingContext(),
      isScrollable: isScrollable(),
      containingBlock: computedStyle.position === 'absolute' || computedStyle.position === 'fixed' ? 'viewport' : 'parent'
    },
    eventListeners: getEventListeners(element),
    parentChain: getParentChain(),
    viewport: {
      scrollX: window.scrollX,
      scrollY: window.scrollY,
      innerWidth: window.innerWidth,
      innerHeight: window.innerHeight
    }
  };
}`,
		},
		{
			Name: "componentTree",
			Metadata: ScriptMetadata{
				Description: "Display the React/Vue component hierarchy and structure for debugging component relationships",
				Category:    "debugging",
				Tags:        []string{"react", "vue", "components", "hierarchy"},
				Author:      "Brummer",
				Version:     "1.0.0",
				Examples: []string{
					"componentTree()",
					"componentTree(document.querySelector('#app'))",
				},
				Parameters: map[string]string{
					"rootElement": "Optional root element to start from (defaults to document.body)",
				},
				ReturnType: "Object with component tree structure and debugging info",
			},
			Code: `function componentTree(rootElement = document.body) {
  const getComponentInfo = (element) => {
    const info = {
      tagName: element.tagName?.toLowerCase(),
      id: element.id || null,
      classes: element.classList ? Array.from(element.classList) : [],
      componentType: null,
      props: null,
      state: null,
      hooks: null,
      children: []
    };

    // React component detection
    const reactFiber = element._reactInternalFiber || 
                      element._reactInternalInstance ||
                      Object.keys(element).find(key => key.startsWith('__reactInternalInstance'));
                      
    if (reactFiber) {
      const fiber = typeof reactFiber === 'string' ? element[reactFiber] : reactFiber;
      if (fiber) {
        info.componentType = 'React';
        info.reactFiber = {
          type: fiber.type?.name || fiber.elementType?.name || 'Unknown',
          key: fiber.key,
          props: fiber.memoizedProps || fiber.pendingProps,
          state: fiber.memoizedState,
          hooks: fiber.hooks || 'Not accessible'
        };
      }
    }

    // Vue component detection
    if (element.__vue__ || element._vnode) {
      info.componentType = 'Vue';
      const vueInstance = element.__vue__ || element._vnode?.context;
      if (vueInstance) {
        info.vueComponent = {
          name: vueInstance.$options?.name || vueInstance.constructor.name,
          props: vueInstance.$props,
          data: vueInstance.$data,
          computed: Object.keys(vueInstance.$options?.computed || {}),
          methods: Object.keys(vueInstance.$options?.methods || {})
        };
      }
    }

    // Vue 3 detection
    if (element.__vueParentComponent) {
      info.componentType = 'Vue 3';
      const component = element.__vueParentComponent;
      info.vue3Component = {
        type: component.type?.name || 'Anonymous',
        props: component.props,
        setupState: component.setupState,
        ctx: component.ctx
      };
    }

    // Angular detection (basic)
    if (element.ng || element.__ngContext__) {
      info.componentType = 'Angular';
      info.angularContext = 'Detected (use Angular DevTools for details)';
    }

    // Svelte detection
    if (element.__svelte_meta) {
      info.componentType = 'Svelte';
      info.svelteComponent = element.__svelte_meta;
    }

    return info;
  };

  const buildTree = (element, depth = 0) => {
    if (depth > 20) return { error: 'Max depth reached' }; // Prevent infinite recursion
    
    const componentInfo = getComponentInfo(element);
    
    // Process children
    const children = Array.from(element.children || []);
    componentInfo.children = children.map(child => buildTree(child, depth + 1));
    
    return componentInfo;
  };

  const tree = buildTree(rootElement);
  
  // Add summary statistics
  const getStats = (node) => {
    let total = 1;
    let byType = {};
    
    if (node.componentType) {
      byType[node.componentType] = (byType[node.componentType] || 0) + 1;
    }
    
    if (node.children) {
      node.children.forEach(child => {
        const childStats = getStats(child);
        total += childStats.total;
        Object.keys(childStats.byType).forEach(type => {
          byType[type] = (byType[type] || 0) + childStats.byType[type];
        });
      });
    }
    
    return { total, byType };
  };

  const stats = getStats(tree);
  
  return {
    tree,
    stats,
    summary: {
      totalElements: stats.total,
      componentFrameworks: Object.keys(stats.byType),
      componentCounts: stats.byType
    },
    helpers: {
      findByType: (type) => {
        const results = [];
        const search = (node) => {
          if (node.componentType === type) results.push(node);
          if (node.children) node.children.forEach(search);
        };
        search(tree);
        return results;
      },
      findByName: (name) => {
        const results = [];
        const search = (node) => {
          const componentName = node.reactFiber?.type || 
                               node.vueComponent?.name || 
                               node.vue3Component?.type;
          if (componentName && componentName.toLowerCase().includes(name.toLowerCase())) {
            results.push(node);
          }
          if (node.children) node.children.forEach(search);
        };
        search(tree);
        return results;
      }
    }
  };
}`,
		},
		{
			Name: "traceEvents",
			Metadata: ScriptMetadata{
				Description: "Trace and log all events of a specific type with detailed information for debugging event handling",
				Category:    "debugging",
				Tags:        []string{"events", "debugging", "monitoring"},
				Author:      "Brummer",
				Version:     "1.0.0",
				Examples: []string{
					"traceEvents('click')",
					"traceEvents('scroll', document.querySelector('#content'))",
					"const stop = traceEvents('resize'); /* later: */ stop()",
				},
				Parameters: map[string]string{
					"eventType": "Type of event to trace (click, scroll, resize, etc.)",
					"target":    "Optional target element (defaults to document)",
					"options":   "Optional event listener options",
				},
				ReturnType: "Function to stop tracing",
			},
			Code: `function traceEvents(eventType, target = document, options = { capture: true }) {
  console.log('🔍 Starting event trace for:', eventType);
  
  const events = [];
  const startTime = performance.now();
  
  const handler = (event) => {
    const timestamp = performance.now();
    const eventInfo = {
      type: event.type,
      timestamp: timestamp,
      relativeTime: timestamp - startTime,
      target: {
        tagName: event.target?.tagName?.toLowerCase(),
        id: event.target?.id || null,
        classes: event.target?.classList ? Array.from(event.target.classList) : [],
        textContent: event.target?.textContent?.substring(0, 50) + '...' || null
      },
      currentTarget: {
        tagName: event.currentTarget?.tagName?.toLowerCase(),
        id: event.currentTarget?.id || null,
        classes: event.currentTarget?.classList ? Array.from(event.currentTarget.classList) : []
      },
      eventPhase: ['', 'CAPTURING_PHASE', 'AT_TARGET', 'BUBBLING_PHASE'][event.eventPhase] || event.eventPhase,
      bubbles: event.bubbles,
      cancelable: event.cancelable,
      defaultPrevented: event.defaultPrevented,
      isTrusted: event.isTrusted
    };

    // Add event-specific details
    if (event.type === 'click' || event.type === 'mousedown' || event.type === 'mouseup') {
      eventInfo.mouse = {
        button: event.button,
        buttons: event.buttons,
        clientX: event.clientX,
        clientY: event.clientY,
        screenX: event.screenX,
        screenY: event.screenY,
        ctrlKey: event.ctrlKey,
        altKey: event.altKey,
        shiftKey: event.shiftKey,
        metaKey: event.metaKey
      };
    }

    if (event.type === 'keydown' || event.type === 'keyup' || event.type === 'keypress') {
      eventInfo.keyboard = {
        key: event.key,
        code: event.code,
        keyCode: event.keyCode,
        ctrlKey: event.ctrlKey,
        altKey: event.altKey,
        shiftKey: event.shiftKey,
        metaKey: event.metaKey
      };
    }

    if (event.type === 'scroll') {
      eventInfo.scroll = {
        scrollTop: event.target.scrollTop,
        scrollLeft: event.target.scrollLeft,
        scrollHeight: event.target.scrollHeight,
        scrollWidth: event.target.scrollWidth,
        clientHeight: event.target.clientHeight,
        clientWidth: event.target.clientWidth
      };
    }

    if (event.type === 'resize') {
      eventInfo.resize = {
        innerWidth: window.innerWidth,
        innerHeight: window.innerHeight,
        outerWidth: window.outerWidth,
        outerHeight: window.outerHeight
      };
    }

    if (event.type.startsWith('touch')) {
      eventInfo.touch = {
        touches: event.touches?.length || 0,
        targetTouches: event.targetTouches?.length || 0,
        changedTouches: event.changedTouches?.length || 0
      };
    }

    events.push(eventInfo);
    
    // Log with styling
    console.group('📍 Event:', event.type, '@', Math.round(eventInfo.relativeTime) + 'ms');
    console.log('Target:', eventInfo.target);
    console.log('Phase:', eventInfo.eventPhase);
    console.log('Details:', eventInfo);
    console.groupEnd();
    
    // Keep only last 100 events to prevent memory issues
    if (events.length > 100) {
      events.shift();
    }
  };

  // Add event listener
  target.addEventListener(eventType, handler, options);
  
  console.log('✅ Event tracing active for', eventType, 'on', target);
  
  // Return function to stop tracing
  const stopTracing = () => {
    target.removeEventListener(eventType, handler, options);
    console.log('🛑 Stopped tracing', eventType, '- captured', events.length, 'events');
    
    // Return summary
    return {
      eventType,
      totalEvents: events.length,
      duration: performance.now() - startTime,
      events: events,
      summary: {
        targets: [...new Set(events.map(e => e.target.tagName))],
        phases: [...new Set(events.map(e => e.eventPhase))],
        avgTimeBetween: events.length > 1 ? 
          (events[events.length - 1].relativeTime - events[0].relativeTime) / (events.length - 1) : 0
      }
    };
  };
  
  // Add helper methods to the stop function
  stopTracing.getEvents = () => events;
  stopTracing.getStats = () => ({
    count: events.length,
    duration: performance.now() - startTime,
    avgFrequency: events.length / ((performance.now() - startTime) / 1000)
  });
  
  return stopTracing;
}`,
		},
		{
			Name: "getBoundingBoxTree",
			Metadata: ScriptMetadata{
				Description: "Get the bounding box tree for layout debugging, showing element boundaries and layout relationships",
				Category:    "layout",
				Tags:        []string{"layout", "bounding-box", "debugging", "css"},
				Author:      "Brummer",
				Version:     "1.0.0",
				Examples: []string{
					"getBoundingBoxTree()",
					"getBoundingBoxTree(document.querySelector('#content'))",
					"getBoundingBoxTree(null, { highlightOverflows: true })",
				},
				Parameters: map[string]string{
					"rootElement": "Optional root element (defaults to document.body)",
					"options":     "Options object { maxDepth: 10, highlightOverflows: false, minSize: 1 }",
				},
				ReturnType: "Object with bounding box tree and layout analysis",
			},
			Code: `function getBoundingBoxTree(rootElement = document.body, options = {}) {
  const opts = {
    maxDepth: 10,
    highlightOverflows: false,
    minSize: 1,
    ...options
  };

  const getLayoutInfo = (element) => {
    const rect = element.getBoundingClientRect();
    const style = window.getComputedStyle(element);
    
    // Check for potential layout issues
    const issues = [];
    
    // Element too small to be useful
    if (rect.width < opts.minSize || rect.height < opts.minSize) {
      issues.push('Element too small');
    }
    
    // Element extends beyond viewport
    if (rect.right > window.innerWidth || rect.bottom > window.innerHeight) {
      issues.push('Extends beyond viewport');
    }
    
    // Element has negative coordinates
    if (rect.x < 0 || rect.y < 0) {
      issues.push('Negative positioning');
    }
    
    // Overflow issues
    const hasScrollableOverflow = style.overflow === 'scroll' || style.overflow === 'auto';
    const hasHiddenOverflow = style.overflow === 'hidden';
    if (element.scrollWidth > element.clientWidth || element.scrollHeight > element.clientHeight) {
      if (!hasScrollableOverflow && !hasHiddenOverflow) {
        issues.push('Content overflow without handling');
      }
    }
    
    // Z-index without positioning
    if (style.zIndex !== 'auto' && style.position === 'static') {
      issues.push('Z-index on static element');
    }
    
    return {
      tagName: element.tagName?.toLowerCase(),
      id: element.id || null,
      classes: element.classList ? Array.from(element.classList) : [],
      bounds: {
        x: Math.round(rect.x * 100) / 100,
        y: Math.round(rect.y * 100) / 100,
        width: Math.round(rect.width * 100) / 100,
        height: Math.round(rect.height * 100) / 100,
        top: Math.round(rect.top * 100) / 100,
        right: Math.round(rect.right * 100) / 100,
        bottom: Math.round(rect.bottom * 100) / 100,
        left: Math.round(rect.left * 100) / 100
      },
      style: {
        position: style.position,
        display: style.display,
        visibility: style.visibility,
        opacity: style.opacity,
        zIndex: style.zIndex,
        overflow: style.overflow,
        transform: style.transform !== 'none' ? style.transform : null
      },
      scrollInfo: {
        scrollWidth: element.scrollWidth,
        scrollHeight: element.scrollHeight,
        clientWidth: element.clientWidth,
        clientHeight: element.clientHeight,
        scrollTop: element.scrollTop,
        scrollLeft: element.scrollLeft
      },
      issues,
      children: []
    };
  };

  const buildTree = (element, depth = 0) => {
    if (depth > opts.maxDepth) {
      return { error: 'Max depth reached' };
    }
    
    const layoutInfo = getLayoutInfo(element);
    
    // Process visible children only
    const children = Array.from(element.children || []).filter(child => {
      const childStyle = window.getComputedStyle(child);
      return childStyle.display !== 'none' && childStyle.visibility !== 'hidden';
    });
    
    layoutInfo.children = children.map(child => buildTree(child, depth + 1));
    
    return layoutInfo;
  };

  const tree = buildTree(rootElement);
  
  // Analyze the tree for common layout issues
  const analyzeTree = (node) => {
    const analysis = {
      totalElements: 0,
      issueCount: 0,
      issueTypes: {},
      largestElement: null,
      smallestElement: null,
      deepestNesting: 0
    };
    
    const traverse = (n, depth = 0) => {
      analysis.totalElements++;
      analysis.deepestNesting = Math.max(analysis.deepestNesting, depth);
      
      if (n.issues?.length > 0) {
        analysis.issueCount++;
        n.issues.forEach(issue => {
          analysis.issueTypes[issue] = (analysis.issueTypes[issue] || 0) + 1;
        });
      }
      
      const area = n.bounds.width * n.bounds.height;
      if (!analysis.largestElement || area > analysis.largestElement.area) {
        analysis.largestElement = { ...n, area };
      }
      if (!analysis.smallestElement || area < analysis.smallestElement.area) {
        analysis.smallestElement = { ...n, area };
      }
      
      if (n.children) {
        n.children.forEach(child => traverse(child, depth + 1));
      }
    };
    
    traverse(node);
    return analysis;
  };

  const analysis = analyzeTree(tree);
  
  // Optional: highlight elements with overflow issues
  if (opts.highlightOverflows) {
    const highlightOverflows = (node) => {
      if (node.issues?.includes('Content overflow without handling')) {
        const element = document.querySelector(
          node.id ? '#' + node.id : 
          node.classes.length ? '.' + node.classes[0] :
          node.tagName
        );
        if (element) {
          element.style.outline = '2px solid red';
          setTimeout(() => element.style.outline = '', 3000);
        }
      }
      if (node.children) {
        node.children.forEach(highlightOverflows);
      }
    };
    highlightOverflows(tree);
  }

  return {
    tree,
    analysis,
    viewport: {
      width: window.innerWidth,
      height: window.innerHeight,
      scrollX: window.scrollX,
      scrollY: window.scrollY
    },
    helpers: {
      findByIssue: (issueType) => {
        const results = [];
        const search = (node) => {
          if (node.issues?.includes(issueType)) {
            results.push(node);
          }
          if (node.children) node.children.forEach(search);
        };
        search(tree);
        return results;
      },
      findOverflowing: () => {
        const results = [];
        const search = (node) => {
          if (node.scrollInfo.scrollWidth > node.scrollInfo.clientWidth ||
              node.scrollInfo.scrollHeight > node.scrollInfo.clientHeight) {
            results.push(node);
          }
          if (node.children) node.children.forEach(search);
        };
        search(tree);
        return results;
      },
      findOffscreen: () => {
        const results = [];
        const search = (node) => {
          if (node.bounds.right < 0 || node.bounds.left > window.innerWidth ||
              node.bounds.bottom < 0 || node.bounds.top > window.innerHeight) {
            results.push(node);
          }
          if (node.children) node.children.forEach(search);
        };
        search(tree);
        return results;
      }
    }
  };
}`,
		},
		{
			Name: "findLayoutIssues",
			Metadata: ScriptMetadata{
				Description: "Comprehensive layout issue detection that requires less context than screenshots for debugging CSS problems",
				Category:    "layout",
				Tags:        []string{"layout", "css", "debugging", "issues"},
				Author:      "Brummer",
				Version:     "1.0.0",
				Examples: []string{
					"findLayoutIssues()",
					"findLayoutIssues({ checkAccessibility: true })",
					"findLayoutIssues({ rootElement: document.querySelector('#main') })",
				},
				Parameters: map[string]string{
					"options": "Options: { rootElement, checkAccessibility: false, checkPerformance: false }",
				},
				ReturnType: "Object with categorized layout issues and recommendations",
			},
			Code: `function findLayoutIssues(options = {}) {
  const opts = {
    rootElement: document.body,
    checkAccessibility: false,
    checkPerformance: false,
    ...options
  };

  const issues = {
    critical: [],
    warning: [],
    info: [],
    performance: [],
    accessibility: []
  };

  const checkElement = (element) => {
    const rect = element.getBoundingClientRect();
    const style = window.getComputedStyle(element);
    const tagName = element.tagName?.toLowerCase();
    
    // Get element identifier for reporting
    const getElementId = (el) => {
      if (el.id) return '#' + el.id;
      if (el.className) return '.' + Array.from(el.classList)[0];
      return el.tagName?.toLowerCase() || 'unknown';
    };
    
    const elementId = getElementId(element);

    // Critical Issues
    
    // 1. Elements with zero dimensions that should have content
    if ((rect.width === 0 || rect.height === 0) && 
        element.textContent?.trim() && 
        style.display !== 'none') {
      issues.critical.push({
        type: 'Zero dimensions with content',
        element: elementId,
        description: 'Element has content but zero width or height',
        suggestion: 'Check CSS display, position, or sizing properties',
        bounds: rect
      });
    }

    // 2. Overlapping elements (z-index issues)
    const siblings = Array.from(element.parentElement?.children || []);
    siblings.forEach(sibling => {
      if (sibling !== element && sibling.tagName) {
        const siblingRect = sibling.getBoundingClientRect();
        const siblingStyle = window.getComputedStyle(sibling);
        
        // Check for overlap
        const overlap = !(rect.right <= siblingRect.left || 
                         rect.left >= siblingRect.right || 
                         rect.bottom <= siblingRect.top || 
                         rect.top >= siblingRect.bottom);
        
        if (overlap && style.position !== 'static' && siblingStyle.position !== 'static') {
          const zIndex1 = parseInt(style.zIndex) || 0;
          const zIndex2 = parseInt(siblingStyle.zIndex) || 0;
          
          if (zIndex1 === zIndex2) {
            issues.warning.push({
              type: 'Overlapping positioned elements',
              element: elementId,
              sibling: getElementId(sibling),
              description: 'Positioned elements overlap with same z-index',
              suggestion: 'Set different z-index values or adjust positioning'
            });
          }
        }
      }
    });

    // 3. Text too small to read
    const fontSize = parseFloat(style.fontSize);
    if (fontSize < 12 && element.textContent?.trim()) {
      issues.warning.push({
        type: 'Text too small',
        element: elementId,
        fontSize: fontSize + 'px',
        description: 'Font size below 12px may be hard to read',
        suggestion: 'Increase font-size for better readability'
      });
    }

    // 4. Content overflow without proper handling
    const hasOverflow = element.scrollWidth > element.clientWidth || 
                       element.scrollHeight > element.clientHeight;
    if (hasOverflow && style.overflow === 'visible') {
      issues.warning.push({
        type: 'Unhandled content overflow',
        element: elementId,
        description: 'Content overflows container without scroll or hidden',
        suggestion: 'Set overflow: auto, scroll, or hidden',
        scrollDimensions: {
          scrollWidth: element.scrollWidth,
          scrollHeight: element.scrollHeight,
          clientWidth: element.clientWidth,
          clientHeight: element.clientHeight
        }
      });
    }

    // 5. Flexbox/Grid issues
    if (style.display === 'flex' || style.display === 'inline-flex') {
      const children = Array.from(element.children);
      
      // Check for flex items with conflicting sizing
      children.forEach(child => {
        const childStyle = window.getComputedStyle(child);
        if (childStyle.flexShrink === '0' && childStyle.minWidth === 'auto') {
          issues.info.push({
            type: 'Flex item sizing issue',
            element: getElementId(child),
            parent: elementId,
            description: 'Flex item with flex-shrink: 0 but no explicit min-width',
            suggestion: 'Set min-width or allow flex-shrink'
          });
        }
      });
    }

    // 6. CSS Grid issues
    if (style.display === 'grid' || style.display === 'inline-grid') {
      // Check for implicit grid items
      const gridTemplateColumns = style.gridTemplateColumns;
      const gridTemplateRows = style.gridTemplateRows;
      const children = Array.from(element.children);
      
      if (gridTemplateColumns === 'none' && children.length > 0) {
        issues.info.push({
          type: 'Implicit grid usage',
          element: elementId,
          description: 'Grid container without explicit grid-template-columns',
          suggestion: 'Define grid-template-columns for predictable layout'
        });
      }
    }

    // Performance Issues
    if (opts.checkPerformance) {
      // 7. Expensive CSS properties
      const expensiveProps = {
        'box-shadow': style.boxShadow !== 'none',
        'border-radius': style.borderRadius !== '0px',
        'transform': style.transform !== 'none',
        'filter': style.filter !== 'none',
        'backdrop-filter': style.backdropFilter !== 'none'
      };
      
      const activeExpensive = Object.entries(expensiveProps)
        .filter(([prop, active]) => active)
        .map(([prop]) => prop);
      
      if (activeExpensive.length > 3) {
        issues.performance.push({
          type: 'Multiple expensive CSS properties',
          element: elementId,
          properties: activeExpensive,
          description: 'Element uses multiple GPU-intensive CSS properties',
          suggestion: 'Consider reducing visual effects or using will-change'
        });
      }

      // 8. Large images without optimization
      if (tagName === 'img') {
        const naturalWidth = element.naturalWidth;
        const naturalHeight = element.naturalHeight;
        const displayWidth = rect.width;
        const displayHeight = rect.height;
        
        if (naturalWidth > displayWidth * 2 || naturalHeight > displayHeight * 2) {
          issues.performance.push({
            type: 'Oversized image',
            element: elementId,
            description: 'Image is much larger than display size',
            natural: { width: naturalWidth, height: naturalHeight },
            display: { width: displayWidth, height: displayHeight },
            suggestion: 'Use appropriately sized images or responsive images'
          });
        }
      }
    }

    // Accessibility Issues
    if (opts.checkAccessibility) {
      // 9. Missing alt text on images
      if (tagName === 'img' && !element.getAttribute('alt')) {
        issues.accessibility.push({
          type: 'Missing alt text',
          element: elementId,
          description: 'Image missing alt attribute',
          suggestion: 'Add descriptive alt text for screen readers'
        });
      }

      // 10. Low color contrast (basic check)
      const color = style.color;
      const backgroundColor = style.backgroundColor;
      if (color !== backgroundColor && element.textContent?.trim()) {
        // This is a simplified check - real contrast calculation is more complex
        issues.accessibility.push({
          type: 'Potential contrast issue',
          element: elementId,
          description: 'Text color and background may have low contrast',
          colors: { color, backgroundColor },
          suggestion: 'Verify color contrast meets WCAG guidelines'
        });
      }

      // 11. Interactive elements without proper focus indication
      const isInteractive = ['button', 'a', 'input', 'select', 'textarea'].includes(tagName) ||
                           element.getAttribute('tabindex') !== null ||
                           element.getAttribute('role') === 'button';
      
      if (isInteractive && style.outline === 'none' && !style.boxShadow?.includes('focus')) {
        issues.accessibility.push({
          type: 'Missing focus indicator',
          element: elementId,
          description: 'Interactive element may not show focus state',
          suggestion: 'Add visible focus styling for keyboard navigation'
        });
      }
    }
  };

  // Traverse DOM tree
  const traverse = (element) => {
    checkElement(element);
    Array.from(element.children || []).forEach(traverse);
  };

  traverse(opts.rootElement);

  // Generate summary and recommendations
  const totalIssues = issues.critical.length + issues.warning.length + 
                     issues.info.length + issues.performance.length + 
                     issues.accessibility.length;

  const summary = {
    total: totalIssues,
    critical: issues.critical.length,
    warning: issues.warning.length,
    info: issues.info.length,
    performance: issues.performance.length,
    accessibility: issues.accessibility.length
  };

  const recommendations = [];
  
  if (issues.critical.length > 0) {
    recommendations.push('🚨 Address critical layout issues first - these likely break functionality');
  }
  if (issues.warning.length > 5) {
    recommendations.push('⚠️ Many layout warnings detected - review CSS structure');
  }
  if (issues.performance.length > 0) {
    recommendations.push('⚡ Performance issues found - consider optimizing CSS and images');
  }
  if (issues.accessibility.length > 0) {
    recommendations.push('♿ Accessibility improvements needed for better user experience');
  }
  if (totalIssues === 0) {
    recommendations.push('✅ No obvious layout issues detected');
  }

  return {
    issues,
    summary,
    recommendations,
    metadata: {
      checkedAt: new Date().toISOString(),
      rootElement: getElementId(opts.rootElement),
      checksPerformed: {
        basic: true,
        accessibility: opts.checkAccessibility,
        performance: opts.checkPerformance
      }
    },
    helpers: {
      getByCriticality: (level) => issues[level] || [],
      getByType: (type) => {
        const allIssues = [...issues.critical, ...issues.warning, ...issues.info, 
                          ...issues.performance, ...issues.accessibility];
        return allIssues.filter(issue => issue.type === type);
      },
      exportReport: () => {
        return JSON.stringify({ issues, summary, recommendations }, null, 2);
      }
    }
  };
}`,
		},
	}
}

// InstallBuiltinScripts installs all built-in scripts to the scripts directory
func InstallBuiltinScripts() error {
	scriptsDir, err := getScriptsDirectory()
	if err != nil {
		return err
	}

	builtinScripts := GetBuiltinScripts()

	for _, script := range builtinScripts {
		filename := script.Name + ".ts"
		filePath := filepath.Join(scriptsDir, filename)

		// Check if script already exists
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("Builtin script %s already exists, skipping...\n", script.Name)
			continue
		}

		// Save the script
		if err := saveScript(script.Name, script.Code, script.Metadata); err != nil {
			fmt.Printf("Warning: failed to install builtin script %s: %v\n", script.Name, err)
			continue
		}

		fmt.Printf("Installed builtin script: %s\n", script.Name)
	}

	return nil
}
