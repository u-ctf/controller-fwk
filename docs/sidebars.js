// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'index',
      label: 'Overview',
    },
    {
      type: 'doc',
      id: 'getting-started',
      label: 'Getting Started',
    },
    {
      type: 'category',
      label: 'Advanced Usage',
      collapsed: false,
      items: [
        'context',
        'dependencies',
        'resources',
        'instrumentation',
        'watcher-interface',
      ],
    },
  ],
};

module.exports = sidebars;
