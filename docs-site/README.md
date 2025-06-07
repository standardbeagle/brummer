# Brummer Documentation Site

This is the documentation site for Brummer, built with [Docusaurus](https://docusaurus.io/).

## Local Development

```bash
cd docs-site
npm install
npm start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

## Build

```bash
npm run build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the main branch.

### Manual Deployment

```bash
npm run deploy
```

## Adding Documentation

1. Add markdown files to the `docs/` directory
2. Update `sidebars.js` if adding new sections
3. Follow the existing structure and frontmatter format

## Structure

- `docs/` - Documentation markdown files
- `src/` - React components and pages
- `static/` - Static assets like images
- `docusaurus.config.js` - Site configuration
- `sidebars.js` - Sidebar navigation structure