/**
 * IE兼容性检测和警告显示
 * 支持IE10及以上版本
 */
(function () {
  'use strict';

  // IE检测函数
  function detectIE() {
    var ua = window.navigator.userAgent;
    var old_ie = ua.indexOf('MSIE ');
    var new_ie = ua.indexOf('Trident/');
    return old_ie > -1 || new_ie > -1;
  }

  // 创建警告内容
  function createWarningContent() {
    var content = [
      '如果你看到这里的文本，那么说明你可能在使用旧版 IE 浏览器。',
      '这意味着在升级浏览器前，你将无法访问此页面。',
      '海豹核心的运行并不依赖 Chrome，但是其 WebUI 需要新版浏览器才能访问',
      '-------------',
      '有三个方案可以解决此问题：',
      '1. 升级本机上的浏览器，推荐使用 Chrome 或新版 Edge 浏览器（推荐）',
      '2. 在自己电脑上完成配置，然后将配置好的目录打包，上传到服务器运行',
      '3. 在自己电脑上，通过 http://IP地址:3211 来访问海豹。如果服务器运营商有防火墙 (如腾讯云轻量服务器)，自行登录操作放行。',
    ];
    return content;
  }

  // 设置页面标题
  function setPageTitle() {
    var title = 'SealDice 海豹核心';
    document.title = title;
  }

  // 显示IE警告
  function showIEWarning() {
    var warningDiv = document.getElementById('ie-warn');
    if (!warningDiv) {
      return;
    }

    var content = createWarningContent();
    var html = '';

    for (var i = 0; i < content.length; i++) {
      var line = content[i];
      if (line.indexOf('1. 升级本机上的浏览器') === 0) {
        html += '<p style="color: #990000">' + line + '</p>';
      } else {
        html += '<p>' + line + '</p>';
      }
    }

    warningDiv.innerHTML = html;
    warningDiv.style.display = 'block';
  }

  // 初始化函数
  function init() {
    // 设置页面标题
    setPageTitle();

    if (detectIE()) {
      // 确保DOM加载完成后再执行
      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', showIEWarning);
      } else {
        showIEWarning();
      }
    }
  }

  // 立即执行初始化
  init();
})();
