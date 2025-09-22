import{_ as a,c as n,o as i,a8 as p}from"./chunks/framework.bOCt8wBo.js";const c=JSON.parse('{"title":"编写敏感词库","description":"","frontmatter":{"lang":"zh-cn","title":"编写敏感词库"},"headers":[],"relativePath":"advanced/edit_sensitive_words.md","filePath":"advanced/edit_sensitive_words.md","lastUpdated":1758531130000}'),l={name:"advanced/edit_sensitive_words.md"};function t(e,s,h,k,r,d){return i(),n("div",null,[...s[0]||(s[0]=[p(`<h1 id="编写敏感词库" tabindex="-1">编写敏感词库 <a class="header-anchor" href="#编写敏感词库" aria-label="Permalink to &quot;编写敏感词库&quot;">​</a></h1><div class="info custom-block"><p class="custom-block-title">本节内容</p><p>本节将介绍敏感词库的编写，请善用侧边栏和搜索，按需阅读文档。</p></div><h2 id="创建文本格式的敏感词库" tabindex="-1">创建文本格式的敏感词库 <a class="header-anchor" href="#创建文本格式的敏感词库" aria-label="Permalink to &quot;创建文本格式的敏感词库&quot;">​</a></h2><p>你可以直接按照以下格式书写 <code>&lt;words&gt;.txt</code>：</p><div class="language-text vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">text</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span>#notice</span></span>
<span class="line"><span>提醒级词汇 1</span></span>
<span class="line"><span>提醒级词汇 2</span></span>
<span class="line"><span></span></span>
<span class="line"><span>#caution</span></span>
<span class="line"><span>注意级词汇 1</span></span>
<span class="line"><span>注意级词汇 2</span></span>
<span class="line"><span></span></span>
<span class="line"><span>#warning</span></span>
<span class="line"><span>警告级词汇</span></span>
<span class="line"><span></span></span>
<span class="line"><span>#danger</span></span>
<span class="line"><span>危险级词汇</span></span></code></pre></div><h2 id="创建-toml-格式的敏感词库" tabindex="-1">创建 TOML 格式的敏感词库 <a class="header-anchor" href="#创建-toml-格式的敏感词库" aria-label="Permalink to &quot;创建 TOML 格式的敏感词库&quot;">​</a></h2><div class="info custom-block"><p class="custom-block-title">TOML 格式</p><p>我们假定你已了解 TOML 格式。如果你对 TOML 还很陌生，可以阅读以下教程或自行在互联网搜索：</p><ul><li><a href="https://toml.io/cn/v1.0.0" target="_blank" rel="noreferrer">TOML 文档</a>、<a href="https://zhuanlan.zhihu.com/p/348057345" target="_blank" rel="noreferrer">TOML 教程</a></li></ul></div><p>你可以直接按照以下格式书写 <code>&lt;words&gt;.toml</code>：</p><div class="language-toml vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">toml</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 元信息，用于填写一些额外的展示内容</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">[</span><span style="--shiki-light:#6F42C1;--shiki-dark:#B392F0;">meta</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 词库名称</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">name = </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;测试词库&#39;</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 作者，和 authors 存在一个即可</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">author = </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;&#39;</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 作者（多个），和 author 存在一个即可</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">authors = [ </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;&lt;匿名&gt;&#39;</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;"> ]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 版本，建议使用语义化版本号</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">version = </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;1.0&#39;</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 简介</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">desc = </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;一个测试词库&#39;</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 协议</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">license = </span><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">&#39;CC-BY-NC-SA 4.0&#39;</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 创建日期，使用 RFC 3339 格式</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">date = </span><span style="--shiki-light:#005CC5;--shiki-dark:#79B8FF;">2023-10-30</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 更新日期，使用 RFC 3339 格式</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">updateDate = </span><span style="--shiki-light:#005CC5;--shiki-dark:#79B8FF;">2023-10-30</span></span>
<span class="line"></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 词表，出现相同词汇时按最高级别判断</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">[</span><span style="--shiki-light:#6F42C1;--shiki-dark:#B392F0;">words</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 忽略级词表，没有实际作用</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">ignore = []</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 提醒级词表</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">notice = [</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;提醒级词汇 1&#39;</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">,</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;提醒级词汇 2&#39;</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 注意级词表</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">caution = [</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;注意级词汇 1&#39;</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">,</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;注意级词汇 2&#39;</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 警告级词表</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">warning = [</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;警告级词汇 1&#39;</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">,</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;警告级词汇 2&#39;</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;"># 危险级词表</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">danger = [</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;危险级词汇 1&#39;</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">,</span></span>
<span class="line"><span style="--shiki-light:#032F62;--shiki-dark:#9ECBFF;">  &#39;危险级词汇 2&#39;</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">]</span></span></code></pre></div>`,9)])])}const o=a(l,[["render",t]]);export{c as __pageData,o as default};
