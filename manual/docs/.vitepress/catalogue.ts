export const deployNav = {
  text: "部署",
  base: "",
  items: [
    {
      text: "基础知识（电脑小白先看我）",
      items: [
        { text: "计算机相关", link: "/deploy/about_pc" },
        { text: "开源程序相关", link: "/deploy/about_opensource" },
      ],
    },
    {
      text: "部署指南",
      items: [
        { text: "快速开始", link: "/deploy/quick-start" },
        { text: "特色功能", link: "/deploy/special_feature" },
        { text: "海豹的本地文件", link: "/deploy/about_file" },
        { text: "迁移", link: "/deploy/transfer" },
        { text: "数据库检查和修复", link: "/deploy/db-repair" },
      ],
    },
    {
      text: "安卓相关",
      items: [
        { text: "安卓海豹常见问题", link: "/deploy/android" },
        { text: "配置安卓端保活", link: "/deploy/android_keepalive"},
      ],
    },
    {
      text: "连接平台",
      items: [
        { text: "QQ", link: "/deploy/platform-qq" },
        { text: "QQ - Docker 中的海豹", link: "/deploy/platform-qq-docker" },
        { text: "KOOK", link: "/deploy/platform-kook" },
        { text: "DoDo", link: "/deploy/platform-dodo" },
        { text: "Discord", link: "/deploy/platform-discord" },
        { text: "Telegram", link: "/deploy/platform-telegram" },
        { text: "Slack", link: "/deploy/platform-slack" },
        { text: "Minecraft", link: "/deploy/platform-minecraft" },
        { text: "钉钉", link: "/deploy/platform-dingtalk" },
      ],
    },
  ],
}

export const deploySidebar = {
  text: "部署",
  base: "",
  items: [
    {
      text: "基础知识（电脑小白先看我）",
      items: [
        { text: "计算机相关", link: "/deploy/about_pc" },
        { text: "开源程序相关", link: "/deploy/about_opensource" },
      ],
    },
    {
      text: "部署指南",
      items: [
        { text: "快速开始", link: "/deploy/quick-start" },
        { text: "特色功能", link: "/deploy/special_feature" },
        { text: "海豹的本地文件", link: "/deploy/about_file" },
        { text: "迁移", link: "/deploy/transfer" },
        { text: "数据库检查和修复", link: "/deploy/db-repair" },
      ],
    },
    {
      text: "安卓相关",
      items: [
        { text: "安卓海豹常见问题", link: "/deploy/android" },
        { text: "配置安卓端保活",
          link: "/deploy/android_keepalive" ,
          items: [
            { text: "授予海豹核心必要权限", link: "/deploy/android_keepalive#%E6%8E%88%E4%BA%88%E6%B5%B7%E8%B1%B9%E6%A0%B8%E5%BF%83%E5%BF%85%E8%A6%81%E6%9D%83%E9%99%90" },
            { text: "HyperOS", link: "/deploy/android_keepalive#hyperos" },
            { text: "MIUI", link: "/deploy/android_keepalive#miui" },
            { text: "鸿蒙系统", link: "/deploy/android_keepalive#%E9%B8%BF%E8%92%99%E7%B3%BB%E7%BB%9F" },
            { text: "ColorOS", link: "/deploy/android_keepalive#coloros" },
            { text: "通用设置", link: "/deploy/android_keepalive#%E9%80%9A%E7%94%A8%E8%AE%BE%E7%BD%AE" },
          ],
        },
      ],
    },
    {
      text: "连接平台",
      items: [
        {
          text: "QQ",
          link: "/deploy/platform-qq",
          items: [
            { text: "前言", link: "/deploy/platform-qq#%E5%89%8D%E8%A8%80" },
            { text: "内置客户端", link: "/deploy/platform-qq#%E5%86%85%E7%BD%AE%E5%AE%A2%E6%88%B7%E7%AB%AF" },
            { text: "Lagrange", link: "/deploy/platform-qq#lagrange" },
            { text: "LLOneBot", link: "/deploy/platform-qq#llonebot" },
            { text: "NapCatQQ", link: "/deploy/platform-qq#napcatqq" },
            { text: "Shamrock", link: "/deploy/platform-qq#shamrock" },
            { text: "Shamrock LSPatch", link: "/deploy/platform-qq#shamrock-lspatch" },
            { text: "Chronocat", link: "/deploy/platform-qq#chronocat" },
            { text: "官方机器人", link: "/deploy/platform-qq#%E5%AE%98%E6%96%B9%E6%9C%BA%E5%99%A8%E4%BA%BA" },
          ]
        },
        { text: "QQ - Docker 中的海豹", link: "/deploy/platform-qq-docker" },
        { text: "KOOK", link: "/deploy/platform-kook" },
        { text: "DoDo", link: "/deploy/platform-dodo" },
        { text: "Discord", link: "/deploy/platform-discord" },
        { text: "Telegram", link: "/deploy/platform-telegram" },
        { text: "Slack", link: "/deploy/platform-slack" },
        { text: "Minecraft", link: "/deploy/platform-minecraft" },
        { text: "钉钉", link: "/deploy/platform-dingtalk" },
      ],
    },
  ],
}

export const config = {
  text: "配置",
  base: "",
  items: [
    {
      text: "扩展功能",
      items: [
        { text: "自定义文案", link: "/config/custom_text" },
        { text: "自定义回复", link: "/config/reply" },
        { text: "牌堆", link: "/config/deck" },
        { text: "JavaScript 插件", link: "/config/jsscript" },
        { text: "帮助文档", link: "/config/helpdoc" },
        { text: "拦截", link: "/config/censor" },
      ],
    },
    {
      text: "综合设置",
      items: [
        { text: "黑白名单", link: "/config/ban" },
        { text: "备份", link: "/config/backup" },
        { text: "自动退出不活跃群组", link: "/config/quit_grp_auto" },
      ],
    },
  ],
}

export const useNav = {
  text: "使用",
  base: "",
  items: [
    {
      text: "新手入门",
      items: [
        { text: "基础概念", link: "/use/introduce" },
        { text: "快速上手", link: "/use/quick-start" },
      ],
    },
    {
      text: "核心指令",
      items: [
        { text: "核心指令", link: "/use/core" },
      ],
    },
    {
      text: "规则扩展",
      items: [
        { text: "克苏鲁的呼唤 7 版", link: "/use/coc7" },
        { text: "龙与地下城 5E", link: "/use/dnd5e" },
        { text: "属性同义词", link: "/use/attribute_alias" },
        { text: "其它规则支持", link: "/use/other_rules" },
      ],
    },
    {
      text: "功能扩展",
      items: [
        { text: "故事", link: "/use/story" },
        { text: "日志", link: "/use/log" },
        { text: "功能", link: "/use/fun" },
        { text: "牌堆和自定义回复", link: "/use/deck_and_reply" },
      ],
    },
    {
      text: "常见问题",
      items: [
        { text: "常见问题", link: "/use/faq" },
      ],
    },
  ],
}

export const useSidebar = {
  text: "使用",
  base: "",
  items: [
    {
      text: "新手入门",
      items: [
        { text: "基础概念", link: "/use/introduce" },
        { text: "快速上手", link: "/use/quick-start" },
      ],
    },
    {
      text: "核心指令",
      items: [
        { text: "核心指令", link: "/use/core" },
      ],
    },
    {
      text: "规则扩展",
      items: [
        { text: "克苏鲁的呼唤 7 版", link: "/use/coc7" },
        { text: "龙与地下城 5E", link: "/use/dnd5e" },
        { text: "属性同义词", link: "/use/attribute_alias" },
        {
          text: "其它规则支持",
          link: "/use/other_rules",
          items: [
            { text: "绿色三角洲 Delta Green", link: "/use/other_rules#%E7%BB%BF%E8%89%B2%E4%B8%89%E8%A7%92%E6%B4%B2-delta-green" },
            { text: "黑暗世界 World of Darkness", link: "/use/other_rules#%E9%BB%91%E6%9A%97%E4%B8%96%E7%95%8C-world-of-darkness" },
            { text: "双重十字 Double Cross", link: "/use/other_rules#%E5%8F%8C%E9%87%8D%E5%8D%81%E5%AD%97-double-cross" },
            { text: "暗影狂奔 Shadowrun", link: "/use/other_rules#%E6%9A%97%E5%BD%B1%E7%8B%82%E5%A5%94-shadowrun" },
            { text: "共鸣性怪异 Emoklore", link: "/use/other_rules#%E5%85%B1%E9%B8%A3%E6%80%A7%E6%80%AA%E5%BC%82-emoklore" },
          ]
        },
      ],
    },
    {
      text: "功能扩展",
      items: [
        { text: "故事", link: "/use/story" },
        { text: "日志", link: "/use/log" },
        { text: "功能", link: "/use/fun" },
        { text: "牌堆和自定义回复", link: "/use/deck_and_reply" },
      ],
    },
    {
      text: "常见问题",
      items: [
        { text: "常见问题", link: "/use/faq" },
      ],
    },
  ],
}

export const advanced = {
  text: "进阶",
  base: "",
  items: [
    { text: "进阶介绍", link: "/advanced/introduce" },
    { text: "内置脚本语言", link: "/advanced/script" },
    {
      text: "扩展功能进阶",
      items: [
        { text: "编写复杂文案", link: "/advanced/edit_complex_custom_text" },
        { text: "编写自定义回复", link: "/advanced/edit_reply" },
        { text: "编写牌堆", link: "/advanced/edit_deck" },
        { text: "编写帮助文档", link: "/advanced/edit_helpdoc" },
        { text: "编写敏感词库", link: "/advanced/edit_sensitive_words" },
      ],
    },
    {
      text: "Javascript 插件",
      items: [
        { text: "前言", link: "/advanced/js_start" },
        { text: "常见用法示例", link: "/advanced/js_example" },
        { text: "编写新的 TRPG 规则", link: "/advanced/js_gamesystem" },
        { text: "API 列表", link: "/advanced/js_api_list" },
      ],
    },
  ],
}

export const about = {
  text: "关于",
  base: "",
  items: [
    { text: "从零开始", link: "/about/start-from-zero" },
    { text: "关于", link: "/about/about" },
    { text: "许可协议", link: "/about/license" },
    { text: "参与项目", link: "/about/develop" },
    { text: "归档", link: "/about/archieve" },
  ],
}
