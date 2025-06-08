/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  tutorialSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started',
        'installation',
        'quick-start',
      ],
    },
    {
      type: 'category',
      label: 'User Guide',
      items: [
        'user-guide/navigation',
        'user-guide/process-management',
        'user-guide/log-management',
        'user-guide/settings',
      ],
    },
    {
      type: 'category',
      label: 'Features',
      items: [
        'features/multi-package-support',
        'features/intelligent-monitoring',
        'features/error-detection',
        'features/url-detection',
      ],
    },
    {
      type: 'category',
      label: 'MCP Integration',
      items: [
        'mcp-integration/overview',
        'mcp-integration/api-reference',
        'mcp-integration/client-setup',
        'mcp-integration/events',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'development/architecture',
        'development/building',
        'development/contributing',
      ],
    },
  ],
};

export default sidebars;