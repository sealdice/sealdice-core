// @ts-ignore
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
      {
        text: "平台",
        children: [
          "platform-qq.md",
          "platform-kook.md",
          "platform-dodo.md",
          "platform-discord.md",
          "platform-slack.md"
        ],
      },
    ],
  },
  {
    text: "配置",
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
      {
        text: "常见问题",
        children: ["faq.md"],
      },
    ],
  },
  {
    text: "进阶",
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
  {
    text: "关于",
    prefix: "/about/",
    children: ["about.md", "license.md", "develop.md"],
  },
]);
