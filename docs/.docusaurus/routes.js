import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/__docusaurus/debug',
    component: ComponentCreator('/__docusaurus/debug', '5ff'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/config',
    component: ComponentCreator('/__docusaurus/debug/config', '5ba'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/content',
    component: ComponentCreator('/__docusaurus/debug/content', 'a2b'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/globalData',
    component: ComponentCreator('/__docusaurus/debug/globalData', 'c3c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/metadata',
    component: ComponentCreator('/__docusaurus/debug/metadata', '156'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/registry',
    component: ComponentCreator('/__docusaurus/debug/registry', '88c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/routes',
    component: ComponentCreator('/__docusaurus/debug/routes', '000'),
    exact: true
  },
  {
    path: '/docs',
    component: ComponentCreator('/docs', '74f'),
    routes: [
      {
        path: '/docs',
        component: ComponentCreator('/docs', '44b'),
        routes: [
          {
            path: '/docs',
            component: ComponentCreator('/docs', '80e'),
            routes: [
              {
                path: '/docs/advanced-topics/custom-models',
                component: ComponentCreator('/docs/advanced-topics/custom-models', '015'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/advanced-topics/model-packs',
                component: ComponentCreator('/docs/advanced-topics/model-packs', '00a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/advanced-topics/model-settings',
                component: ComponentCreator('/docs/advanced-topics/model-settings', 'bf7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/cli-reference',
                component: ComponentCreator('/docs/cli-reference', '29e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/core-concepts/branches',
                component: ComponentCreator('/docs/core-concepts/branches', '190'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/core-concepts/context-management',
                component: ComponentCreator('/docs/core-concepts/context-management', 'bdd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/core-concepts/plans',
                component: ComponentCreator('/docs/core-concepts/plans', '8fa'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/installation',
                component: ComponentCreator('/docs/installation', '057'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/introduction',
                component: ComponentCreator('/docs/introduction', 'b43'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/quick-start',
                component: ComponentCreator('/docs/quick-start', '6b8'),
                exact: true,
                sidebar: "docs"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
