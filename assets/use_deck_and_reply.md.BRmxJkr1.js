import{_ as c,C as s,c as r,o as l,a8 as o,j as t,G as n,a as d}from"./chunks/framework.bOCt8wBo.js";const w=JSON.parse('{"title":"牌堆 自定义回复","description":"","frontmatter":{"lang":"zh-cn","title":"牌堆 自定义回复"},"headers":[],"relativePath":"use/deck_and_reply.md","filePath":"use/deck_and_reply.md","lastUpdated":1758531130000}'),i={name:"use/deck_and_reply.md"},p={class:"info custom-block"},_={class:"info custom-block"},u={class:"info custom-block"};function h(m,e,k,f,b,E){const a=s("ChatBox");return l(),r("div",null,[e[3]||(e[3]=o('<h1 id="牌堆-自定义回复" tabindex="-1">牌堆 自定义回复 <a class="header-anchor" href="#牌堆-自定义回复" aria-label="Permalink to &quot;牌堆 自定义回复&quot;">​</a></h1><div class="info custom-block"><p class="custom-block-title">本节内容</p><p>本节将展示牌堆和自定义回复相关的指令，请善用侧边栏和搜索，按需阅读文档。</p></div><div class="tip custom-block"><p class="custom-block-title">提示：如何自定义？</p><p>牌堆和自定义回复都是海豹提供的扩展性功能，此处只展示相关控制指令，如果你想知道如何进行自定义，请转到 <a href="./../advanced/introduce.html">进阶介绍</a>。</p></div><h2 id="draw-抽牌-管理牌堆" tabindex="-1"><code>.draw</code> 抽牌 / 管理牌堆 <a class="header-anchor" href="#draw-抽牌-管理牌堆" aria-label="Permalink to &quot;`.draw` 抽牌 / 管理牌堆&quot;">​</a></h2><p>关于牌堆功能的一般性介绍，请参阅 <a href="./../config/deck.html">配置 - 牌堆</a>。</p><p><code>.draw help</code> 显示帮助信息。</p><h3 id="信息查询" tabindex="-1">信息查询 <a class="header-anchor" href="#信息查询" aria-label="Permalink to &quot;信息查询&quot;">​</a></h3><p><code>.draw list</code> 列出当前装载的牌堆列表。</p><p><code>.draw keys &lt;牌堆&gt;</code> 查看特定牌堆可抽取的牌组列表。</p><p><code>.draw search &lt;牌组名称&gt;</code> 模糊搜索指定牌组。</p><p><code>.draw desc &lt;牌组名称&gt;</code> 查看牌组所属牌堆的详细信息。</p>',11)),t("div",p,[e[0]||(e[0]=t("p",{class:"custom-block-title"},"示例",-1)),n(a,{messages:[{content:".draw list",send:!0},{content:`载入并开启的牌堆:
- GRE单词 格式: Dice! 作者:于言诺 版本:1.0.1 牌组数量: 1
- IELTS单词 格式: Dice! 作者:于言诺 版本:1.0.1 牌组数量: 1
- TOEFL单词 格式: Dice! 作者:于言诺 版本:1.0.1 牌组数量: 1
- SealDice内置牌堆 格式: Dice! 作者:<因过长略去> 版本:1.2.0 牌组数量: 8`},{content:".draw keys GRE单词",send:!0},{content:`牌组关键字列表:
GRE单词`},{content:".draw search 单词",send:!0},{content:`找到以下牌组:
- GRE单词
- TOEFL单词
- IELTS单词`},{content:".draw desc GRE单词",send:!0},{content:`牌堆信息:
牌堆: GRE单词
格式: Dice!
作者: 于言诺
版本: 1.0.1
牌组数量: 1
时间: 2022/5/23
更新时间: 2022/8/16
牌组: GRE单词`}]})]),e[4]||(e[4]=o('<p>需要说明，在以上的例子中，「GRE单词」同时是牌堆名与牌组名。在 <code>.draw keys GRE单词</code> 中，它作为牌堆名出现；在 <code>.draw desc GRE单词</code> 中，它作为牌组名出现。</p><p><code>.draw keys</code> 列出所有可抽取的牌组列表。</p><div class="warning custom-block"><p class="custom-block-title">注意：谨慎使用</p><p>这一指令会将<strong>所有</strong>可抽取的牌组列出，在牌组较多时造成刷屏。</p><p>如果你不希望列出所有可抽取的牌堆，可以修改自定义回复中的<code>其他:抽牌_列表</code>，将其中的内容替换为你想要展示的牌组列表即可。</p></div><h3 id="抽牌" tabindex="-1">抽牌 <a class="header-anchor" href="#抽牌" aria-label="Permalink to &quot;抽牌&quot;">​</a></h3><p><code>.draw &lt;牌组名称&gt; (&lt;数量&gt;#)</code> 在指定牌组抽指定数量的牌，默认为抽 1 张。</p>',5)),t("div",_,[e[1]||(e[1]=t("p",{class:"custom-block-title"},"示例",-1)),n(a,{messages:[{content:".draw GRE单词 3#",send:!0},{content:`<木落>抽出了：
GRE3178
invoice n.
发票, 发货单, 货物。`},{content:`<木落>抽出了：
GRE4889
rig n.
索具装备, 钻探设备, 钻探平台, 钻塔。`},{content:`<木落>抽出了：
GRE5421
austerity n.
严峻, 严厉, 朴素, 节俭, 苦行。`}]})]),e[5]||(e[5]=t("p",null,[d("当指定的牌组名称不存在时，将会进行模糊搜索，效果与 "),t("code",null,"draw search"),d(" 类似。")],-1)),t("div",u,[e[2]||(e[2]=t("p",{class:"custom-block-title"},"示例",-1)),n(a,{messages:[{content:".draw 单词",send:!0},{content:`找不到这个牌组，但发现一些相似的:
- GRE单词
- TOEFL单词
- IELTS单词`}]})]),e[6]||(e[6]=o('<h3 id="自定义回复引用牌堆" tabindex="-1">自定义回复引用牌堆 <a class="header-anchor" href="#自定义回复引用牌堆" aria-label="Permalink to &quot;自定义回复引用牌堆&quot;">​</a></h3><p>对于想要在自定义文案或自定义回复中引用牌堆的骰主，可使用 <code>#{DRAW-关键词}</code> 进行调用。</p><h3 id="牌堆管理" tabindex="-1">牌堆管理 <a class="header-anchor" href="#牌堆管理" aria-label="Permalink to &quot;牌堆管理&quot;">​</a></h3><p><code>.draw reload</code> 重新加载牌堆，仅限 Master 使用。</p><h2 id="reply-管理自定义回复" tabindex="-1"><code>.reply</code> 管理自定义回复 <a class="header-anchor" href="#reply-管理自定义回复" aria-label="Permalink to &quot;`.reply` 管理自定义回复&quot;">​</a></h2><p>关于自定义回复功能的一般性介绍，请参阅 <a href="./../config/reply.html">配置 - 自定义回复</a>。</p><p><code>.reply (on|off)</code> 开启、关闭本群的自定义回复功能。</p><p>以上指令等价于 <code>.ext reply (on|off)</code>。</p>',8))])}const R=c(i,[["render",h]]);export{w as __pageData,R as default};
