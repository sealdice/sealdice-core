// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export default sidebar({
  "/deploy/": [
    {
      text: "部署",
      link: "/deploy/",
      prefix: "/deploy/",
      children: [
        "quick-start.md",
        "transfer.md",
        "special_feature.md",
        {
          text: "平台",
          children: [
            "platform-qq.md",
            "platform-kook.md",
            "platform-dodo.md",
            "Platform-slack.md"
          ],
        },
      ],
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
        {
          text: "新手入门",
          children: [
            "introduce.md",
            "quick-start.md",
          ],
        },
        {
          text: "核心指令",
          children: ["core.md"],
        },
        {
          text: "规则扩展",
          children: [
            "coc7.md",
            "dnd5e.md",
            "attribute_alias.md",
            "other_rules.md",
          ],
        },
        {
          text: "功能扩展",
          children: [
            "story.md",
            "log.md",
            "fun.md",
            "deck_and_reply.md",
          ],
        },
        "faq.md",
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
        {
          text: "扩展功能进阶",
          children: [
            "edit_complex_custom_text.md",
            "edit_reply.md",
            "edit_deck.md",
            "edit_jsscript.md",
            "edit_helpdoc.md",
            "edit_sensitive_words.md",
          ],
        },
      ],
    },
  ],
  "/about/": [
    {
      text: "关于",
      link: "/about/",
      prefix: "/about/",
      children: ["about.md", "license.md", "develop.md"],
    },
  ],
});
