import { navbar } from "vuepress-theme-hope";

export default navbar([
  {
    text: "首页",
    link: "/",
  },
  {
    text: "部署",
    prefix: "/deploy/",
    children: [
      "quick-start.md",
      "transfer.md",
      "special_feature.md",
    ],
  },
  {
    text: "配置",
    prefix: "/config/",
    children: [
      "custom_text.md",
      {
        text: "扩展功能",
        children: [
          "reply.md",
          "deck.md",
          "jsscript.md",
          "helpdoc.md",
          "censor.md",
        ],
      },
      {
        text: "综合设置",
        children: ["ban.md"],
      },
    ],
  },
  {
    text: "使用",
    prefix: "/use/",
    children: [
      "introduce.md",
      "quick-start.md",
      "faq.md",
      {
        text: "核心指令",
        children: ["core.md"],
      },
      {
        text: "扩展指令",
        children: [
          "coc7.md",
          "dnd5e.md",
          "story.md",
          "log.md",
          "fun.md",
          "deck_and_reply.md",
          "sr.md",
          "attribute_alias.md",
        ],
      },
    ],
  },
  {
    text: "进阶",
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
  {
    text: "项目",
    prefix: "/develop/",
    children: ["about.md", "develop.md"],
  },
]);
