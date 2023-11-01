import { hopeTheme } from "vuepress-theme-hope";
import navbar from "./navbar";
import sidebar from "./sidebar";

export default hopeTheme({
  hostname: "http://localhost:8080",
  author: {
    name: "SealDice Team",
    url: "https://github.com/sealdice",
  },
  favicon: "/images/sealdice.svg",
  docsDir: "docs",

  navbar,
  logo: "/images/sealdice.svg",
  repo: "sealdice/sealdice-core",

  sidebar,

  breadcrumb: true,
  pageInfo: ["ReadingTime"],
  contributors: false,
  editLink: false,
  docsRepo: "sealdice/sealdice-manual-next",
  docsBranch: "main",
  displayFooter: false,
  home: "/index.md",
  pure: true,
  print: false,

  iconAssets: "iconfont",

  metaLocales: {
    editLink: "在 GitHub 上编辑此页",
  },

  plugins: {
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
