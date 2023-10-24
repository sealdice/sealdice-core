import { hopeTheme } from "vuepress-theme-hope";
import navbar from "./navbar";
import sidebar from "./sidebar";

export default hopeTheme({
  hostname: "http://localhost:8080",
  author: {
    name: "SealDice Team",
    url: "https://github.com/sealdice",
  },
  favicon: "/images/sealdice.ico",
  docsDir: "docs",

  navbar,
  logo: "/images/sealdice.ico",
  repo: "sealdice/sealdice-core",

  sidebar,

  breadcrumb: false,
  pageInfo: ["ReadingTime"],
  contributors: false,
  editLink: false,
  docsRepo: "sealdice/sealdice-manual-next",
  docsBranch: "main",
  displayFooter: false,
  home: "/index.md",
  pure: true,
  print: false,

  metaLocales: {
    editLink: "在 Github 上编辑此页",
  },

  plugins: {
    autoCatalog: false,
    copyCode: {
      showInMobile: true,
    },
    mdEnhance: {
      container: true,
      tabs: true,
      figure: true,
      imgLazyload: true,
      imgMark: true,
      imgSize: true,
      align: true,
      mermaid: true,
    },
  },
});
