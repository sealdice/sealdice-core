import { defineComponent as _, createElementBlock as r, openBlock as o, normalizeStyle as b, createElementVNode as t, renderSlot as V, normalizeClass as L, toDisplayString as v, createCommentVNode as q, mergeDefaults as H, createBlock as Q, withCtx as M, createVNode as ne, shallowRef as C, withModifiers as re, withDirectives as ie, resolveDynamicComponent as de, createTextVNode as ce, vShow as ue, isRef as me, watch as z, shallowReadonly as pe, toValue as fe, getCurrentScope as ge, onScopeDispose as ve, onMounted as _e, onBeforeUnmount as he, unref as k, Fragment as F, renderList as U } from "vue";
const ye = { class: "qq-header__title" }, $e = /* @__PURE__ */ _({
  __name: "QHeader",
  props: {
    height: { default: 45 }
  },
  setup(a) {
    return (e, s) => (o(), r("header", {
      class: "qq-header",
      style: b({ "--qq-header-height": e.height + "px" })
    }, [
      s[0] || (s[0] = t("div", { class: "qq-header__btn-bar" }, [
        t("span", { class: "qq-header__btn red" }),
        t("span", { class: "qq-header__btn yellow" }),
        t("span", { class: "qq-header__btn green" })
      ], -1)),
      t("div", ye, [
        V(e.$slots, "default", {}, void 0, !0)
      ])
    ], 4));
  }
}), B = (a, e) => {
  const s = a.__vccOpts || a;
  for (const [u, c] of e)
    s[u] = c;
  return s;
}, Ce = /* @__PURE__ */ B($e, [["__scopeId", "data-v-48fc79dd"]]), qe = { class: "qq_container" }, we = /* @__PURE__ */ _({
  name: "QMain",
  __name: "QMain",
  setup(a) {
    return (e, s) => (o(), r("main", qe, [
      V(e.$slots, "default", {}, void 0, !0)
    ]));
  }
}), be = /* @__PURE__ */ B(we, [["__scopeId", "data-v-0ee2b41a"]]);
var X = /* @__PURE__ */ ((a) => (a.sage_green = "sage_green", a.red = "red", a.orange = "orange", a.purple = "purple", a.blue = "blue", a.grey = "grey", a))(X || {});
const w = {
  self: !1,
  avatar: void 0,
  tag: void 0,
  tagColor: void 0,
  isBot: !1
}, Be = { class: "qq-message__avatar-span" }, Ie = {
  key: 1,
  class: "qq-message__text-avatar"
}, ke = { class: "qq-message__user-name nocopy qq-text-ellipsis" }, Se = { class: "qq-text-ellipsis" }, Pe = {
  key: 2,
  class: "qq-bot-label qq-bot-label--middle qq-bot-label--mini"
}, Qe = { class: "message-content__wrapper" }, Ve = /* @__PURE__ */ _({
  __name: "QMessageBase",
  props: {
    self: { type: Boolean, default: w.self },
    name: {},
    avatar: { default: w.avatar },
    tag: { default: w.tag },
    tagColor: { default: X.grey },
    isBot: { type: Boolean, default: w.isBot }
  },
  setup(a) {
    return (e, s) => (o(), r("section", {
      class: L(["message-container", e.self ? ["message-container--self", "message-container--align-right"] : ""])
    }, [
      t("div", Be, [
        e.avatar ? (o(), r("div", {
          key: 0,
          style: b({ backgroundImage: `url(${e.avatar})` }),
          class: "qq-message__avatar"
        }, null, 4)) : (o(), r("div", Ie, [
          t("span", null, v(e.name[0]), 1)
        ]))
      ]),
      t("div", ke, [
        t("span", Se, v(e.name), 1),
        e.tag && typeof e.tagColor == "string" ? (o(), r("div", {
          key: 0,
          class: L(["q-tag qq-message__user-label", [`q-tag--${e.tagColor}`]])
        }, v(e.tag), 3)) : e.tag && typeof e.tagColor == "object" ? (o(), r("div", {
          key: 1,
          class: "q-tag qq-message__user-label",
          style: b({ backgroundColor: e.tagColor.backgroundColor, color: e.tagColor.color })
        }, v(e.tag), 5)) : q("", !0),
        e.isBot ? (o(), r("label", Pe, s[0] || (s[0] = [
          t("i", {
            class: "q-svg-icon q-icon",
            style: { width: "1em", height: "1em" }
          }, [
            t("svg", {
              id: "channle_robot_small_16",
              viewBox: "0 0 16 16",
              fill: "none",
              xmlns: "http://www.w3.org/2000/svg"
            }, [
              t("path", {
                "fill-rule": "evenodd",
                "clip-rule": "evenodd",
                d: "M8.5 3.29119C8.91311 3.10159 9.2 2.6843 9.2 2.2C9.2 1.53726 8.66274 1 8 1C7.33726 1 6.8 1.53726 6.8 2.2C6.8 2.6843 7.08689 3.10159 7.5 3.29119V4.00015V4.02059C4.42023 4.27466 2 6.85472 2 10V10.5715C2 12.465 3.53502 14 5.42857 14H10.5714C12.465 14 14 12.465 14 10.5715V10C14 6.85472 11.5798 4.27466 8.5 4.02059V4.00015V3.29119ZM13 10.5715V10C13 7.23863 10.7614 5.00005 8 5.00005C5.23858 5.00005 3 7.23863 3 10V10.5715C3 11.9127 4.08731 13 5.42857 13H10.5714C11.9127 13 13 11.9127 13 10.5715ZM5.7002 9.5V7.5H6.7002V9.5H5.7002ZM9.2998 7.5V9.5H10.2998V7.5H9.2998Z",
                fill: "currentColor"
              })
            ])
          ], -1)
        ]))) : q("", !0)
      ]),
      t("div", Qe, [
        V(e.$slots, "default", {}, void 0, !0)
      ])
    ], 2));
  }
}), T = /* @__PURE__ */ B(Ve, [["__scopeId", "data-v-7cc2f4cb"]]), Me = { class: "reply-title" }, De = { class: "reply-info" }, Le = { class: "qq-text-ellipsis" }, He = { class: "reply-content" }, Te = {
  key: 0,
  class: "pic"
}, Ae = { class: "image nocopy" }, Re = ["alt", "src"], We = {
  key: 1,
  class: "mixed-container qq-text-ellipsis"
}, Ee = /* @__PURE__ */ _({
  __name: "QReplyBase",
  props: {
    self: { type: Boolean, default: !1 },
    target: {},
    replyText: { default: "" },
    replyImageUrl: { default: void 0 },
    replyImageAlt: { default: void 0 },
    maxImgWidth: { default: "200px" },
    maxImgHeight: { default: "220px" }
  },
  setup(a) {
    return (e, s) => (o(), r("div", {
      class: L(["reply-element nocopy", e.self ? ["reply-element--self"] : ""])
    }, [
      t("div", Me, [
        t("div", De, [
          t("span", Le, v(e.target), 1)
        ])
      ]),
      t("div", He, [
        e.replyImageUrl ? (o(), r("div", Te, [
          t("div", Ae, [
            t("img", {
              class: "image-content",
              style: b({ "--max-image-width": e.maxImgWidth, "--max-image-height": e.maxImgHeight }),
              alt: e.replyImageAlt,
              src: e.replyImageUrl
            }, null, 12, Re)
          ])
        ])) : (o(), r("span", We, v(e.replyText), 1))
      ])
    ], 2));
  }
}), Ne = /* @__PURE__ */ B(Ee, [["__scopeId", "data-v-db622448"]]), ze = { class: "message-content reply-message__inner" }, Fe = /* @__PURE__ */ _({
  __name: "QReply",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    target: {},
    replyText: {},
    replyImageUrl: {},
    replyImageAlt: {},
    maxImgWidth: {},
    maxImgHeight: {}
  }, {
    ...w,
    replyText: "",
    replyImageUrl: void 0,
    replyImageAlt: void 0,
    maxImgWidth: "200px",
    maxImgHeight: "220px"
  }),
  setup(a) {
    return (e, s) => (o(), Q(T, {
      self: e.self,
      name: e.name,
      avatar: e.avatar,
      tag: e.tag,
      "tag-color": e.tagColor,
      "is-bot": e.isBot
    }, {
      default: M(() => [
        t("div", {
          class: L(["msg-content-container reply-message__container", e.self ? "container--self" : "container--others"])
        }, [
          t("div", ze, [
            ne(Ne, {
              self: e.self,
              target: e.target,
              "reply-text": e.replyText,
              "reply-image-url": e.replyImageUrl,
              "reply-image-alt": e.replyImageAlt,
              "max-img-width": e.maxImgWidth,
              "max-img-height": e.maxImgHeight
            }, null, 8, ["self", "target", "reply-text", "reply-image-url", "reply-image-alt", "max-img-width", "max-img-height"]),
            t("span", null, [
              V(e.$slots, "default", {}, void 0, !0)
            ])
          ])
        ], 2)
      ]),
      _: 3
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), Ue = /* @__PURE__ */ B(Fe, [["__scopeId", "data-v-544a0ce6"]]), Ze = /* @__PURE__ */ _({
  __name: "QText",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    maxImgWidth: {},
    maxImgHeight: {}
  }, {
    ...w,
    maxImgWidth: "230px",
    maxImgHeight: "250px"
  }),
  setup(a) {
    return (e, s) => (o(), Q(T, {
      self: e.self,
      name: e.name,
      avatar: e.avatar,
      tag: e.tag,
      "tag-color": e.tagColor,
      "is-bot": e.isBot
    }, {
      default: M(() => [
        t("div", {
          class: L(["msg-content-container mix-message__container", e.self ? "container--self" : "container--others"])
        }, [
          t("div", {
            class: "message-content mix-message__inner",
            style: b({ "--max-image-width": e.maxImgWidth, "--max-image-height": e.maxImgHeight })
          }, [
            t("span", null, [
              V(e.$slots, "default", {}, void 0, !0)
            ])
          ], 4)
        ], 2)
      ]),
      _: 3
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), Ge = /* @__PURE__ */ B(Ze, [["__scopeId", "data-v-b90ccc44"]]), je = ["href", "download"], Oe = { class: "message-content mix-message__inner" }, Je = ["src", "alt"], Ke = {
  key: 0,
  class: "file-info-mask"
}, Xe = {
  key: 0,
  class: "qq-text-ellipsis",
  style: { display: "flex" }
}, Ye = { class: "qq-text-ellipsis" }, xe = { key: 1 }, et = /* @__PURE__ */ _({
  __name: "QImage",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    src: {},
    alt: {},
    isFile: { type: Boolean },
    fileName: {},
    fileSize: {},
    maxWidth: {},
    maxHeight: {},
    canDownload: { type: Boolean }
  }, {
    ...w,
    alt: void 0,
    isFile: !1,
    fileName: void 0,
    fileSize: void 0,
    maxWidth: "230px",
    maxHeight: "250px",
    canDownload: !0
  }),
  setup(a) {
    return (e, s) => (o(), Q(T, {
      self: e.self,
      name: e.name,
      avatar: e.avatar,
      tag: e.tag,
      "tag-color": e.tagColor,
      "is-bot": e.isBot
    }, {
      default: M(() => [
        t("div", {
          class: L(["msg-content-container mix-message__container mix-message__container--pic", e.self ? "container--self" : "container--others"])
        }, [
          e.isFile && e.canDownload ? (o(), r("a", {
            key: 0,
            class: "file-link",
            href: e.src,
            download: e.fileName
          }, null, 8, je)) : q("", !0),
          t("div", Oe, [
            t("div", {
              class: "image pic-element",
              style: b({ "--max-image-width": e.maxWidth, "--max-image-height": e.maxHeight })
            }, [
              t("img", {
                src: e.src,
                alt: e.alt,
                class: "image-content"
              }, null, 8, Je)
            ], 4),
            e.isFile ? (o(), r("div", Ke, [
              e.fileName ? (o(), r("p", Xe, [
                t("span", Ye, v(e.fileName), 1)
              ])) : q("", !0),
              e.fileSize ? (o(), r("p", xe, v(e.fileSize), 1)) : q("", !0)
            ])) : q("", !0)
          ])
        ], 2)
      ]),
      _: 1
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), tt = /* @__PURE__ */ B(et, [["__scopeId", "data-v-4c3abb39"]]), at = { class: "file-message--content nocopy" }, st = ["href", "download"], ot = { class: "normal-file file-element" }, lt = { class: "file-header" }, nt = { class: "file-name" }, rt = { class: "qq-text-ellipsis" }, it = {
  key: 0,
  class: "file-info"
}, dt = /* @__PURE__ */ _({
  __name: "QFile",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    fileName: {},
    fileSize: {},
    fileSrc: {},
    iconSrc: {},
    canDownload: { type: Boolean }
  }, {
    ...w,
    fileSize: void 0,
    fileSrc: void 0,
    iconSrc: void 0,
    canDownload: !0
  }),
  setup(a) {
    return (e, s) => (o(), Q(T, {
      self: e.self,
      name: e.name,
      avatar: e.avatar,
      tag: e.tag,
      "tag-color": e.tagColor,
      "is-bot": e.isBot
    }, {
      default: M(() => [
        t("div", at, [
          e.fileSrc && e.canDownload ? (o(), r("a", {
            key: 0,
            class: "file-link",
            href: e.fileSrc,
            download: e.fileName
          }, null, 8, st)) : q("", !0),
          t("div", ot, [
            t("div", lt, [
              t("p", nt, [
                t("span", rt, v(e.fileName), 1)
              ]),
              e.iconSrc ? (o(), r("div", {
                key: 0,
                class: "file-icon",
                style: b({ backgroundImage: `url(${e.iconSrc})` })
              }, null, 4)) : q("", !0)
            ]),
            e.fileSize ? (o(), r("div", it, [
              t("span", null, v(e.fileSize), 1)
            ])) : q("", !0)
          ])
        ])
      ]),
      _: 1
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), ct = /* @__PURE__ */ B(dt, [["__scopeId", "data-v-b9e76760"]]), ut = {
  key: 0,
  class: "message__timestamp no-copy"
}, mt = { class: "babble" }, pt = {
  key: 1,
  class: "gray-tip-message no-copy"
}, ft = { class: "gray-tip-content babble" }, gt = /* @__PURE__ */ _({
  __name: "QTip",
  props: {
    isTime: { type: Boolean, default: !1 }
  },
  setup(a) {
    return (e, s) => e.isTime ? (o(), r("div", ut, [
      t("span", mt, [
        V(e.$slots, "default", {}, void 0, !0)
      ])
    ])) : (o(), r("div", pt, [
      t("div", ft, [
        V(e.$slots, "default", {}, void 0, !0)
      ])
    ]));
  }
}), vt = /* @__PURE__ */ B(gt, [["__scopeId", "data-v-bfd44f5c"]]), _t = {
  id: "play_fill_24",
  viewBox: "0 0 24 24",
  fill: "none",
  xmlns: "http://www.w3.org/2000/svg"
}, ht = /* @__PURE__ */ _({
  name: "QVoicePlayIcon",
  __name: "QVoicePlayIcon",
  setup(a) {
    return (e, s) => (o(), r("svg", _t, s[0] || (s[0] = [
      t("path", {
        d: "M6 20.2428V3.75722C6 2.56846 7.31683 1.85207 8.31488 2.49786L21.0537 10.7406C21.9672 11.3317 21.9672 12.6683 21.0537 13.2594L8.31488 21.5021C7.31683 22.1479 6 21.4315 6 20.2428Z",
        fill: "currentColor"
      }, null, -1)
    ])));
  }
}), yt = {
  id: "pause_24",
  viewBox: "0 0 24 24",
  fill: "none",
  xmlns: "http://www.w3.org/2000/svg"
}, $t = /* @__PURE__ */ _({
  name: "QVoicePauseIcon",
  __name: "QVoicePauseIcon",
  setup(a) {
    return (e, s) => (o(), r("svg", yt, s[0] || (s[0] = [
      t("path", {
        "fill-rule": "evenodd",
        "clip-rule": "evenodd",
        d: "M17 4C17.5523 4 18 4.44771 18 5L18 19C18 19.5523 17.5523 20 17 20L15 20C14.4477 20 14 19.5523 14 19L14 5C14 4.44771 14.4477 4 15 4L17 4Z",
        fill: "currentColor"
      }, null, -1),
      t("path", {
        "fill-rule": "evenodd",
        "clip-rule": "evenodd",
        d: "M9 4C9.55229 4 10 4.44771 10 5L10 19C10 19.5523 9.55228 20 9 20L7 20C6.44772 20 6 19.5523 6 19L6 5C6 4.44771 6.44772 4 7 4L9 4Z",
        fill: "currentColor"
      }, null, -1)
    ])));
  }
}), Ct = { class: "ptt-message__inner" }, qt = { class: "ptt-element__top-area" }, wt = { class: "ptt-element__button" }, bt = {
  class: "q-svg-icon q-icon",
  style: { width: "10px", height: "10px" }
}, Bt = { class: "ptt-element__duration" }, It = { class: "ptt-element__bottom-area" }, kt = { class: "ptt-element__bottom-area-text" }, St = /* @__PURE__ */ _({
  __name: "QVoiceBase",
  props: {
    self: { type: Boolean, default: !1 },
    name: {},
    avatar: { default: void 0 },
    tag: { default: void 0 },
    tagColor: { default: void 0 },
    isBot: { type: Boolean, default: !1 },
    src: {},
    text: { default: "[呃，什么都没有听到]" },
    play: {},
    playPaused: { type: Boolean },
    formatedDuration: {}
  },
  setup(a) {
    const e = C(!1);
    let s;
    function u() {
      s = setTimeout(() => {
        e.value = !e.value;
      }, 500);
    }
    function c() {
      clearTimeout(s);
    }
    function m(n) {
      n.preventDefault(), e.value = !e.value;
    }
    return (n, f) => (o(), Q(T, {
      self: n.self,
      name: n.name,
      avatar: n.avatar,
      tag: n.tag,
      "tag-color": n.tagColor,
      "is-bot": n.isBot
    }, {
      default: M(() => [
        t("div", {
          class: L(["msg-content-container mix-message__container ptt-message nocopy", n.self ? "container--self" : "container--others"]),
          onContextmenu: re(m, ["prevent"]),
          onTouchstartPassive: u,
          onTouchend: c,
          onTouchmove: c,
          onTouchcancel: c
        }, [
          t("div", Ct, [
            t("div", {
              class: "ptt-element",
              onClick: f[0] || (f[0] = //@ts-ignore
              (...y) => n.play && n.play(...y))
            }, [
              t("div", qt, [
                t("div", wt, [
                  t("i", bt, [
                    (o(), Q(de(n.playPaused ? ht : $t)))
                  ])
                ]),
                V(n.$slots, "default", {}, void 0, !0),
                t("div", Bt, [
                  t("span", null, v(n.formatedDuration), 1)
                ])
              ])
            ]),
            ie(t("div", It, [
              t("div", kt, [
                ce(v(n.text) + " ", 1),
                t("div", {
                  class: "ptt-element__bottom-area-icon",
                  onClick: f[1] || (f[1] = (y) => e.value = !1)
                }, f[2] || (f[2] = [
                  t("i", {
                    class: "q-svg-icon q-icon",
                    style: { width: "1em", height: "1em" }
                  }, [
                    t("svg", {
                      id: "arrow_up_24",
                      viewBox: "0 0 24 24",
                      fill: "none",
                      xmlns: "http://www.w3.org/2000/svg"
                    }, [
                      t("path", {
                        d: "M3 16L12 7L21 16",
                        stroke: "currentColor",
                        "stroke-width": "1.5",
                        "stroke-linejoin": "round"
                      })
                    ])
                  ], -1)
                ]))
              ])
            ], 512), [
              [ue, e.value]
            ])
          ])
        ], 34)
      ]),
      _: 3
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), Y = /* @__PURE__ */ B(St, [["__scopeId", "data-v-d0f84712"]]);
function j(a) {
  return ge() ? (ve(a), !0) : !1;
}
const O = typeof window < "u" && typeof document < "u";
typeof WorkerGlobalScope < "u" && globalThis instanceof WorkerGlobalScope;
function J(a, e = 1e3, s = {}) {
  const {
    immediate: u = !0,
    immediateCallback: c = !1
  } = s;
  let m = null;
  const n = C(!1);
  function f() {
    m && (clearInterval(m), m = null);
  }
  function y() {
    n.value = !1, f();
  }
  function S() {
    const $ = fe(e);
    $ <= 0 || (n.value = !0, c && a(), f(), n.value && (m = setInterval(a, $)));
  }
  if (u && O && S(), me(e) || typeof e == "function") {
    const $ = z(e, () => {
      n.value && O && S();
    });
    j($);
  }
  return j(y), {
    isActive: pe(n),
    pause: y,
    resume: S
  };
}
function x(a) {
  const e = Math.floor(a / 60), s = Math.round(a % 60);
  return e > 0 ? `${e}'${s}"` : `${s}"`;
}
function ee() {
  const a = C(), e = C(0), s = C(!0), u = C(!0), c = C("");
  let m, n;
  function f(d, g) {
    var P;
    (P = a.value) == null || P.style.setProperty(
      "--process-item-color",
      "var(--qq-text-secondary-01)"
    );
    let h = 0;
    const R = Math.floor(g) / d.length * 1e3, W = Math.floor(g) / 100 * 1e3, { pause: A, resume: i } = J(
      () => {
        var D, E;
        s.value && (A(), h = 0, d.forEach((N) => N.style.removeProperty("--process-item-color")), (D = a.value) == null || D.style.setProperty(
          "--process-item-color",
          "var(--qq-text_primary)"
        )), (E = d[h]) == null || E.style.setProperty("--process-item-color", "var(--qq-text_primary)"), h++;
      },
      R,
      { immediate: !0 }
    ), { pause: l, resume: I } = J(
      () => {
        s.value && (l(), e.value = 0), e.value++;
      },
      W,
      { immediate: !0 }
    );
    m = () => {
      A(), l();
    }, n = () => {
      i(), I();
    };
  }
  function y(d, g) {
    f(d, g), s.value = !1, u.value = !1;
  }
  function S() {
    m == null || m(), u.value = !0;
  }
  function $() {
    n == null || n(), u.value = !1;
  }
  function p() {
    s.value = !0, u.value = !0;
  }
  return {
    progressItemsRef: a,
    processLinePos: e,
    playEnded: s,
    playPaused: u,
    formatedDuration: c,
    /** Call when starting playback from beginning */
    onPlaybackStart: y,
    /** Call to pause (e.g. audioCtx.suspend / audio.pause) */
    pause: S,
    /** Call to resume (e.g. audioCtx.resume / audio.play) */
    resume: $,
    /** Call on playback ended */
    reset: p
  };
}
const Pt = /* @__PURE__ */ _({
  __name: "QVoice",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    src: {},
    text: {},
    volume: {}
  }, {
    ...w,
    text: "[呃，什么都没有听到]",
    volume: 1
  }),
  setup(a) {
    const e = a, {
      progressItemsRef: s,
      processLinePos: u,
      formatedDuration: c,
      playEnded: m,
      playPaused: n,
      onPlaybackStart: f,
      pause: y,
      resume: S,
      reset: $
    } = ee(), p = C([]);
    let d, g, h;
    function R(l) {
      return l = l / 1.2, l < 4 ? 4 : l > 30 ? 30 : l;
    }
    function W(l) {
      return l >= 0 ? 1 : l <= -80 ? 0.2 : (l - -80) / 80 * (1 - 0.2) + 0.2;
    }
    async function A(l) {
      d = new (window.AudioContext || window.webkitAudioContext)(), h = d.createGain(), h.connect(d.destination);
      try {
        const P = await (await fetch(l)).arrayBuffer();
        g = await d.decodeAudioData(P);
        const D = g.getChannelData(0), E = R(g.duration), N = Math.floor(D.length / E), te = Array.from({ length: E }, (Wt, Z) => {
          const G = D.slice(Z * N, (Z + 1) * N), ae = Math.sqrt(G.reduce((oe, le) => oe + le ** 2, 0) / G.length), se = Math.max(ae, 1e-10);
          return 20 * Math.log10(se);
        });
        c.value = x(g.duration), p.value = te.map(W);
      } catch (I) {
        console.error("Error loading audio file:", I), c.value = "Error", p.value = Array(10).fill(0.05);
      }
    }
    function i() {
      if (!(d === void 0 || g === void 0 || s.value === void 0 || h === void 0))
        if (m.value) {
          const l = d.createBufferSource();
          l.buffer = g, l.connect(h), l.onended = () => {
            $();
          }, l.start(), f(
            [...s.value.children],
            g.duration
          );
        } else
          d.state === "running" ? (d.suspend(), y()) : d.state === "suspended" && (d.resume(), S());
    }
    return _e(async () => {
      await A(e.src);
    }), he(() => {
      d !== void 0 && d.close();
    }), z(
      () => e.volume,
      (l) => {
        h !== void 0 && (h.gain.value = l);
      }
    ), (l, I) => (o(), Q(Y, {
      self: l.self,
      name: l.name,
      avatar: l.avatar,
      tag: l.tag,
      "tag-color": l.tagColor,
      "is-bot": l.isBot,
      src: l.src,
      play: i,
      "play-paused": k(n),
      "formated-duration": k(c),
      text: l.text
    }, {
      default: M(() => [
        t("div", {
          ref_key: "progressItemsRef",
          ref: s,
          class: "ptt-element__progress",
          style: { "--process-item-color": "var(--qq-text_primary)" }
        }, [
          k(m) ? q("", !0) : (o(), r("div", {
            key: 0,
            class: "ptt-element__progress-tag",
            style: b({ left: `calc(${k(u)}% - 1px)` })
          }, null, 4)),
          (o(!0), r(F, null, U(p.value, (P, D) => (o(), r("div", {
            key: D,
            class: "ptt-element__progress-item",
            style: b({ height: `${P * 100}%` })
          }, null, 4))), 128))
        ], 512)
      ]),
      _: 1
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot", "src", "play-paused", "formated-duration", "text"]));
  }
}), Qt = ["src"], Vt = /* @__PURE__ */ _({
  __name: "QVoiceLegacy",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    src: {},
    text: {},
    volume: {}
  }, {
    ...w,
    text: "[呃，什么都没有听到]",
    volume: 1
  }),
  setup(a) {
    const e = a, {
      progressItemsRef: s,
      processLinePos: u,
      formatedDuration: c,
      playEnded: m,
      playPaused: n,
      onPlaybackStart: f,
      pause: y,
      resume: S,
      reset: $
    } = ee(), p = C(), d = C(0), g = C([]);
    function h(i) {
      return i = i / 1.2, Array.from({ length: i >= 25 ? 25 : i < 5 ? 5 : i }, () => R(30, 60));
    }
    function R(i, l) {
      return Math.floor(Math.random() * (l - i + 1)) + i;
    }
    async function W() {
      p.value && (d.value = Math.round(p.value.duration), c.value = x(d.value), g.value = h(d.value));
    }
    function A() {
      p.value && s.value && (m.value ? (p.value.play(), f(
        [...s.value.children],
        d.value
      )) : p.value.paused ? (p.value.play(), S()) : (p.value.pause(), y()));
    }
    return z(
      () => e.volume,
      (i) => {
        p.value !== void 0 && (p.value.volume = i);
      }
    ), (i, l) => (o(), Q(Y, {
      self: i.self,
      name: i.name,
      avatar: i.avatar,
      tag: i.tag,
      "tag-color": i.tagColor,
      "is-bot": i.isBot,
      src: i.src,
      play: A,
      "play-paused": k(n),
      "formated-duration": k(c),
      text: i.text
    }, {
      default: M(() => [
        t("audio", {
          ref_key: "audio",
          ref: p,
          src: i.src,
          onEnded: l[0] || (l[0] = //@ts-ignore
          (...I) => k($) && k($)(...I)),
          onLoadedmetadata: W
        }, null, 40, Qt),
        t("div", {
          ref_key: "progressItemsRef",
          ref: s,
          class: "ptt-element__progress",
          style: { "--process-item-color": "var(--qq-text_primary)" }
        }, [
          k(m) ? q("", !0) : (o(), r("div", {
            key: 0,
            class: "ptt-element__progress-tag",
            style: b({ left: `calc(${k(u)}% - 1px)` })
          }, null, 4)),
          (o(!0), r(F, null, U(g.value, (I, P) => (o(), r("div", {
            key: P,
            class: "ptt-element__progress-item",
            style: b({ height: `${I}%` })
          }, null, 4))), 128))
        ], 512)
      ]),
      _: 1
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot", "src", "play-paused", "formated-duration", "text"]));
  }
}), Mt = { class: "forward-msg nocopy" }, Dt = { class: "fwd-title text-ellipsis" }, Lt = { class: "count" }, Ht = /* @__PURE__ */ _({
  __name: "QForward",
  props: /* @__PURE__ */ H({
    self: { type: Boolean },
    name: {},
    avatar: {},
    tag: {},
    tagColor: {},
    isBot: { type: Boolean },
    title: {},
    contents: {}
  }, {
    ...w,
    title: "群聊的聊天记录"
  }),
  setup(a) {
    return (e, s) => (o(), Q(T, {
      self: e.self,
      name: e.name,
      avatar: e.avatar,
      tag: e.tag,
      "tag-color": e.tagColor,
      "is-bot": e.isBot
    }, {
      default: M(() => [
        t("div", Mt, [
          t("div", Dt, v(e.title), 1),
          (o(!0), r(F, null, U(e.contents, (u, c) => (o(), r("div", {
            key: c,
            class: "fwd-content text-ellipsis"
          }, v(u), 1))), 128)),
          t("div", Lt, "查看" + v(e.contents.length) + "条转发消息", 1)
        ])
      ]),
      _: 1
    }, 8, ["self", "name", "avatar", "tag", "tag-color", "is-bot"]));
  }
}), Tt = /* @__PURE__ */ B(Ht, [["__scopeId", "data-v-c71d6830"]]), At = [
  Ue,
  Ge,
  tt,
  ct,
  vt,
  Pt,
  Vt,
  Tt,
  T
], Rt = [Ce, be], K = (a, e) => {
  if (e.__name) {
    a.component(e.__name, e);
    return;
  }
  if (e.name) {
    a.component(e.name, e);
    return;
  }
  console.error("[Fake QQ UI] The following component has no name.", e);
}, Nt = {
  install(a) {
    At.forEach((e) => K(a, e)), Rt.forEach((e) => K(a, e));
  }
};
export {
  Nt as FakeQQUI,
  ct as QFile,
  Tt as QForward,
  Ce as QHeader,
  tt as QImage,
  be as QMain,
  Ue as QReply,
  X as QTagColors,
  Ge as QText,
  vt as QTip,
  Pt as QVoice
};
//# sourceMappingURL=fake-qq-ui.js.map
