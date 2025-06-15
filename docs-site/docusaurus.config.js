// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import {themes as prismThemes} from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Brummer',
  tagline: 'Your Terminal UI Development Buddy with intelligent monitoring',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://standardbeagle.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/brummer/',

  // GitHub pages deployment config.
  organizationName: 'standardbeagle',
  projectName: 'brummer',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/standardbeagle/brummer/tree/main/docs-site/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: 'img/brummer-social-card.jpg',
      navbar: {
        title: 'Brummer',
        logo: {
          alt: 'Brummer Logo',
          src: 'img/bee.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'https://github.com/standardbeagle/brummer',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/docs/getting-started',
              },
              {
                label: 'Installation',
                to: '/docs/installation',
              },
              {
                label: 'MCP Integration',
                to: '/docs/mcp-integration',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/standardbeagle/brummer',
              },
              {
                label: 'Issues',
                href: 'https://github.com/standardbeagle/brummer/issues',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'Browser Extension (Alpha)',
                to: '/docs/browser-extension',
              },
              {
                label: 'API Reference',
                to: '/docs/api-reference',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Brummer. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;