import { sidebar } from "vuepress-theme-hope";

export default sidebar({
  "/deploy/": [
    {
      text: "部署",
      link: "/deploy/",
      prefix: "/deploy/",
      children: ["quick-start.md", "transfer.md"],
    },
  ],
  "/config/": [
    {
      text: "配置",
      link: "/config/",
      prefix: "/config/",
      children: [
        {
          text: "扩展功能",
          children: [
            "custom_text.md",
            "reply.md",
            "deck.md",
            "jsscript.md",
            "helpdoc.md",
            "censor.md",
          ],
        },
        { text: "综合设置", children: ["ban.md"] },
      ],
    },
  ],
  "/use/": [
    {
      text: "使用",
      link: "/use/",
      prefix: "/use/",
      children: [
        "introduce.md",
        "quick-start.md",
        "special_feature.md",
        "faq.md",
        { text: "核心命令", children: ["core.md", "helper.md"] },
        {
          text: "扩展命令",
          children: [
            "coc7.md",
            "dnd5e.md",
            "story.md",
            "log.md",
            "fun.md",
            "deck_and_reply.md",
            "sr.md",
          ],
        },
      ],
    },
  ],
  "/advanced/": [
    {
      text: "进阶",
      link: "/advanced/",
      prefix: "/advanced/",
      children: [
        "introduce.md",
        "script.md",
        "edit_complex_custom_text.md",
        "edit_reply.md",
        "edit_deck.md",
        "edit_jsscript.md",
        "edit_helpdoc.md",
        "edit_sensitive_words.md",
      ],
    },
  ],
  "/develop/": [
    {
      text: "项目",
      link: "/develop/",
      prefix: "/develop/",
      children: ["about.md", "develop.md"],
    },
  ],
});
