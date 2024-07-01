import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import search from "docusaurus-lunr-search"
import redirect from "@docusaurus/plugin-client-redirects"

const config: Config = {
  title: 'Plandex Docs',
  tagline: 'An AI coding engine for large, real-world tasks',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://docs.plandex.ai',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

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
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/', // Serve the docs at the site's root
          editUrl:
            'https://github.com/plandex-ai/plandex/tree/main/docs/',
        },
        blog: false, // Disable the blog
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Replace with your project's social card
    image: 'img/plandex-social-preview.png',
    colorMode: {
      defaultMode: "dark",
    },  
    navbar: {
      title: 'Plandex Docs',
      logo: {
        alt: 'Plandex Logo',
        src: 'img/plandex-logo-thumb.png',
      },
      items: [
        {
          href: 'https://github.com/plandex-ai/plandex',
          label: 'GitHub',
          position: 'right',
        },
        {
          label: 'Discord',
          href: 'https://discord.gg/plandex-ai',
          position: 'right',
        },
        {
          label: 'X',
          href: 'https://x.com/PlandexAI',
          position: 'right',
        },
        {
          label: 'YouTube',
          href: 'https://www.youtube.com/@plandex-ny5ry',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',      
      copyright: `Copyright © ${new Date().getFullYear()} PlandexAI, Inc.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,

  plugins: [
    search,
    [
      '@docusaurus/plugin-client-redirects',
      { redirects: [{ from: '/', to: '/intro'}] },
    ],
  ]
};

export default config;
