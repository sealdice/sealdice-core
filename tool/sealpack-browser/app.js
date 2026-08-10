(function () {
  const defaultBase = "https://repo.sealdice.com/dice/api/store";
  const contentFilters = [
    { key: "", label: "全部" },
    { key: "scripts", label: "脚本" },
    { key: "decks", label: "牌堆" },
    { key: "reply", label: "回复" },
    { key: "helpdoc", label: "帮助文档" },
    { key: "templates", label: "模板" },
  ];

  const state = {
    base: defaultBase,
    content: "",
    category: "",
    search: "",
    sortBy: "updateTime",
    order: "desc",
    pageNum: 1,
    pageSize: 20,
    next: false,
    items: [],
    visibleItems: [],
    active: null,
    allCategories: [],
  };

  const $base = $("#apiBase");
  const $status = $("#backendStatus");
  const $activeCategory = $("#activeCategory");
  const $contentFilters = $("#contentFilters");
  const $storeCategories = $("#storeCategories");
  const $packageList = $("#packageList");
  const $resultTitle = $("#resultTitle");
  const $resultCount = $("#resultCount");
  const $pageInfo = $("#pageInfo");
  const $detailEmpty = $("#detailEmpty");
  const $detailBody = $("#detailBody");

  function joinUrl(base, path) {
    return base.replace(/\/+$/, "") + path;
  }

  function getApiBase() {
    return ($base.val() || defaultBase).trim().replace(/\/+$/, "");
  }

  function setStatus(text, kind) {
    $status.text(text);
    $status.attr("data-kind", kind || "");
  }

  function fmtCount(n) {
    return new Intl.NumberFormat("zh-CN").format(n || 0);
  }

  function escapeHtml(input) {
    return String(input ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function shortText(text, limit) {
    const value = String(text || "").trim();
    return value.length > limit ? value.slice(0, limit) + "…" : value;
  }

  function buildQuery(params) {
    const q = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== "" && value != null) q.set(key, String(value));
    });
    return q.toString();
  }

  function selectedChips() {
    return state.allCategories.map((item) =>
      `<button type="button" class="chip ${state.category === item ? "active" : ""}" data-category="${escapeHtml(item)}">${escapeHtml(item)}</button>`
    ).join("");
  }

  function renderFilters() {
    $contentFilters.html(contentFilters.map((item) =>
      `<button type="button" class="${state.content === item.key ? "active" : ""}" data-content="${escapeHtml(item.key)}">${escapeHtml(item.label)}</button>`
    ).join(""));
    $storeCategories.html(selectedChips() || '<p class="muted">先加载列表再显示分类。</p>');
    $("#baseLabel").text(getApiBase());
    $("#activeCategory").text(state.category || "全部");
  }

  function renderList(items) {
    const tpl = document.getElementById("packageItemTpl");
    $packageList.empty();
    if (!items.length) {
      $packageList.html('<article class="pkg-item"><p class="muted">没有匹配的豹包。</p></article>');
      return;
    }

    items.forEach((item, index) => {
      const node = tpl.content.firstElementChild.cloneNode(true);
      if (state.active && state.active.id === item.id && state.active.version === item.version) {
        node.classList.add("active");
      }
      node.dataset.index = String(index);
      node.querySelector(".pkg-name").textContent = item.name || item.id;
      node.querySelector(".pkg-desc").textContent = shortText(item.description || "无简介", 120);
      node.querySelector(".pkg-version").textContent = item.version || "";
      node.querySelector(".pkg-author").textContent = (item.authors && item.authors.length) ? item.authors.join(" / ") : "匿名";
      const badges = [];
      if (item.contents && item.contents.length) badges.push(item.contents.join(" · "));
      if (item.storeAssets && item.storeAssets.category) badges.push(item.storeAssets.category);
      badges.push(`下载 ${fmtCount(item.download && item.download.downloadCount)}`);
      node.querySelector(".pkg-badges").innerHTML = badges.map((b) => `<span class="badge">${escapeHtml(b)}</span>`).join("");
      node.querySelector(".open-btn").addEventListener("click", () => selectPackage(item));
      const downloadUrl = resolveUrl(item.download && item.download.url);
      const downloadBtn = node.querySelector(".download-btn");
      downloadBtn.setAttribute("href", downloadUrl || "#");
      downloadBtn.setAttribute("target", "_blank");
      downloadBtn.setAttribute("rel", "noopener");
      if (!downloadUrl) {
        downloadBtn.classList.add("disabled");
      }
      node.addEventListener("click", (ev) => {
        if ($(ev.target).is("button")) return;
        if ($(ev.target).is("a")) return;
        selectPackage(item);
      });
      $packageList.append(node);
    });
  }

  function renderDetail(pkg) {
    if (!pkg) {
      $detailEmpty.removeClass("hidden");
      $detailBody.addClass("hidden");
      return;
    }
    $detailEmpty.addClass("hidden");
    $detailBody.removeClass("hidden");
    $("#detailCategory").text((pkg.storeAssets && pkg.storeAssets.category) ? pkg.storeAssets.category : "未分类");
    $("#detailName").text(pkg.name || pkg.id);
    $("#detailDesc").text(pkg.description || "无简介");
    $("#detailAuthors").text((pkg.authors && pkg.authors.length) ? pkg.authors.join(" / ") : "匿名");
    $("#detailVersion").text(pkg.version || "");
    $("#detailContents").text((pkg.contents && pkg.contents.length) ? pkg.contents.join(" / ") : "-");
    $("#detailDownloads").text(fmtCount(pkg.download && pkg.download.downloadCount));
    $("#detailJson").text(JSON.stringify(pkg, null, 2));

    const downloadUrl = resolveUrl(pkg.download && pkg.download.url);
    const zipUrl = pkg.download && pkg.download.zipUrl ? resolveUrl(pkg.download.zipUrl) : "";
    $("#downloadSealpack").attr({ href: downloadUrl || "#", target: "_blank", rel: "noopener" }).toggleClass("disabled", !downloadUrl);
    $("#downloadZip").attr({ href: zipUrl || "#", target: "_blank", rel: "noopener" }).toggleClass("disabled", !zipUrl);
    $("#copyLinkBtn").off("click").on("click", async function () {
      if (!downloadUrl) return;
      await navigator.clipboard.writeText(downloadUrl);
      setStatus("已复制 sealpack 直链", "ok");
      setTimeout(() => setStatus("已就绪", "ok"), 1600);
    });
  }

  function resolveUrl(raw) {
    if (!raw) return "";
    try {
      return new URL(raw, getApiBase() + "/").toString();
    } catch (err) {
      return "";
    }
  }

  function selectPackage(pkg) {
    state.active = pkg;
    renderList();
    renderDetail(pkg);
  }

  function collectCategories(items) {
    const set = new Set();
    items.forEach((item) => {
      if (item.storeAssets && item.storeAssets.category) set.add(item.storeAssets.category);
    });
    state.allCategories = Array.from(set).sort((a, b) => a.localeCompare(b, "zh-Hans-CN"));
  }

  function normalizeResponse(data) {
    if (!data) return [];
    if (Array.isArray(data)) return data;
    if (Array.isArray(data.data)) return data.data;
    if (data.data && Array.isArray(data.data.data)) return data.data.data;
    return [];
  }

  function filterItems(items) {
    const term = state.search.trim().toLowerCase();
    return items.filter((item) => {
      const contentOk = !state.content || (item.contents || []).includes(state.content);
      const categoryOk = !state.category || (item.storeAssets && item.storeAssets.category === state.category);
      const haystack = [
        item.id,
        item.name,
        item.version,
        item.description,
        (item.authors || []).join(" "),
        (item.keywords || []).join(" "),
        (item.contents || []).join(" "),
        item.storeAssets && item.storeAssets.category,
      ].join(" ").toLowerCase();
      const searchOk = !term || haystack.includes(term);
      return contentOk + categoryOk + searchOk === 3;
    });
  }

  function applyViewFilters() {
    const visible = filterItems(state.items);
    state.visibleItems = sortItems(visible);
    renderList(state.visibleItems);
    $resultCount.text(`${state.visibleItems.length} 项`);
    if (state.active) {
      const matched = state.visibleItems.find((item) => state.active && state.active.id === item.id && state.active.version === item.version);
      if (!matched) {
        state.active = state.visibleItems[0] || null;
        renderDetail(state.active);
      }
    }
    if (!state.active && state.visibleItems.length) {
      state.active = state.visibleItems[0];
      renderDetail(state.active);
    }
  }

  function sortItems(items) {
    const { sortBy, order } = state;
    const factor = order === "asc" ? 1 : -1;
    return items.slice().sort((a, b) => {
      const av = sortBy === "name" ? (a.name || a.id || "") : Number(a.download && a.download[sortBy]) || 0;
      const bv = sortBy === "name" ? (b.name || b.id || "") : Number(b.download && b.download[sortBy]) || 0;
      if (typeof av === "string" && typeof bv === "string") {
        return av.localeCompare(bv, "zh-Hans-CN") * factor;
      }
      return (av - bv) * factor;
    });
  }

  function renderPageInfo() {
    $pageInfo.text(`第 ${state.pageNum} 页${state.next ? "，可继续翻页" : "，已到末页"}`);
  }

  async function loadBackendInfo() {
    const base = getApiBase();
    try {
      const info = await $.getJSON(joinUrl(base, "/info"));
      const name = info && info.name ? info.name : "官方源";
      setStatus(`${name} 已连接`, "ok");
    } catch (err) {
      setStatus("连接失败", "error");
    }
  }

  async function loadPage() {
    state.base = getApiBase();
    $("#baseLabel").text(state.base);
    setStatus("加载中…", "busy");

    const params = {
      pageNum: state.pageNum,
      pageSize: state.pageSize,
      content: state.content,
      category: state.category,
      sortBy: state.sortBy,
      order: state.order,
    };

    try {
      const data = await $.getJSON(joinUrl(state.base, "/page") + "?" + buildQuery(params));
      const rawItems = normalizeResponse(data);
      state.items = sortItems(rawItems);
      state.next = Boolean(data && (data.next ?? (data.data && data.data.next)));
      collectCategories(rawItems);
      renderFilters();
      applyViewFilters();
      renderPageInfo();
      $resultTitle.text(state.content ? `${state.content} 豹包` : "全部豹包");
      setStatus("已就绪", "ok");
    } catch (err) {
      $packageList.html('<article class="pkg-item"><p class="muted">接口请求失败。请检查浏览器跨域限制或接口地址。</p></article>');
      setStatus("请求失败", "error");
    }
  }

  function bindEvents() {
    $contentFilters.on("click", "button", function () {
      state.content = $(this).data("content") || "";
      state.pageNum = 1;
      state.active = null;
      renderFilters();
      loadPage();
    });

    $storeCategories.on("click", "button", function () {
      state.category = $(this).data("category") || "";
      state.pageNum = 1;
      state.active = null;
      renderFilters();
      loadPage();
    });

    $("#searchInput").on("input", function () {
      state.search = $(this).val() || "";
      applyViewFilters();
    });

    $("#sortSelect").on("change", function () {
      const [sortBy, order] = String($(this).val()).split("-");
      state.sortBy = sortBy;
      state.order = order;
      loadPage();
    });

    $("#apiBase").on("change", function () {
      state.pageNum = 1;
      loadBackendInfo();
      loadPage();
    });

    $("#refreshBtn").on("click", function () {
      loadBackendInfo();
      loadPage();
    });

    $("#prevPage").on("click", function () {
      if (state.pageNum <= 1) return;
      state.pageNum -= 1;
      loadPage();
    });

    $("#nextPage").on("click", function () {
      if (!state.next) return;
      state.pageNum += 1;
      loadPage();
    });
  }

  function init() {
    renderFilters();
    bindEvents();
    loadBackendInfo();
    loadPage();
  }

  $(init);
})();
