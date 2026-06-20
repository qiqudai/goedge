(function () {
  var FP_COOKIE = "__cdn_guard_fp";
  var BID_COOKIE = "__cdn_guard_bid";
  var FP_MAX_AGE = 30 * 60;
  var BID_MAX_AGE = 30 * 24 * 60 * 60;

  function readCookie(name) {
    var pairs = ("; " + document.cookie).split("; " + name + "=");
    if (pairs.length !== 2) return "";
    return pairs.pop().split(";").shift() || "";
  }

  function cookieSuffix(maxAge) {
    var suffix = "; path=/; max-age=" + maxAge + "; SameSite=Lax";
    if (location.protocol === "https:") suffix += "; Secure";
    return suffix;
  }

  function setCookie(name, value, maxAge) {
    document.cookie = name + "=" + value + cookieSuffix(maxAge);
  }

  function randomHex(bytes) {
    var data = new Uint8Array(bytes);
    if (window.crypto && window.crypto.getRandomValues) {
      window.crypto.getRandomValues(data);
    } else {
      for (var i = 0; i < data.length; i++) data[i] = Math.floor(Math.random() * 256);
    }
    var out = "";
    for (var j = 0; j < data.length; j++) out += ("0" + data[j].toString(16)).slice(-2);
    return out;
  }

  function hash(value) {
    var h1 = 0x811c9dc5;
    var h2 = 0x01000193;
    for (var i = 0; i < value.length; i++) {
      h1 ^= value.charCodeAt(i);
      h1 = Math.imul(h1, 0x01000193);
      h2 ^= value.charCodeAt(i);
      h2 = Math.imul(h2, 0x811c9dc5);
    }
    return ("00000000" + (h1 >>> 0).toString(16)).slice(-8) +
      ("00000000" + (h2 >>> 0).toString(16)).slice(-8);
  }

  function canvasSignal() {
    try {
      var canvas = document.createElement("canvas");
      canvas.width = 240;
      canvas.height = 80;
      var ctx = canvas.getContext("2d");
      ctx.textBaseline = "alphabetic";
      ctx.fillStyle = "#f60";
      ctx.fillRect(8, 8, 64, 24);
      ctx.fillStyle = "#069";
      ctx.font = "17px Arial";
      ctx.fillText("cdn guard 665305", 12, 44);
      ctx.strokeStyle = "rgba(120,20,180,.7)";
      ctx.arc(160, 38, 22, 0, Math.PI * 2, true);
      ctx.stroke();
      return canvas.toDataURL();
    } catch (e) {
      return "canvas:error";
    }
  }

  function webglSignal() {
    try {
      var canvas = document.createElement("canvas");
      var gl = canvas.getContext("webgl") || canvas.getContext("experimental-webgl");
      if (!gl) return "webgl:none";
      var info = gl.getExtension("WEBGL_debug_renderer_info");
      var vendor = info ? gl.getParameter(info.UNMASKED_VENDOR_WEBGL) : gl.getParameter(gl.VENDOR);
      var renderer = info ? gl.getParameter(info.UNMASKED_RENDERER_WEBGL) : gl.getParameter(gl.RENDERER);
      return [vendor, renderer, gl.getParameter(gl.VERSION), gl.getParameter(gl.SHADING_LANGUAGE_VERSION)].join("|");
    } catch (e) {
      return "webgl:error";
    }
  }

  function fontSignal() {
    var fonts = [
      "Arial", "Verdana", "Times New Roman", "Courier New", "Georgia",
      "Microsoft YaHei", "SimSun", "PingFang SC", "Hiragino Sans GB",
      "Noto Sans CJK SC", "Roboto", "Helvetica Neue", "Menlo"
    ];
    var found = [];
    if (document.fonts && document.fonts.check) {
      for (var i = 0; i < fonts.length; i++) {
        if (document.fonts.check("12px \"" + fonts[i] + "\"")) found.push(fonts[i]);
      }
      return found.join(",");
    }
    try {
      var canvas = document.createElement("canvas");
      var ctx = canvas.getContext("2d");
      var base = "monospace";
      ctx.font = "72px " + base;
      var baseWidth = ctx.measureText("mmmmmmmmmmlli").width;
      for (var j = 0; j < fonts.length; j++) {
        ctx.font = "72px \"" + fonts[j] + "\"," + base;
        if (ctx.measureText("mmmmmmmmmmlli").width !== baseWidth) found.push(fonts[j]);
      }
      return found.join(",");
    } catch (e) {
      return "fonts:error";
    }
  }

  function buildFingerprint() {
    var nav = window.navigator || {};
    var screenInfo = window.screen || {};
    return hash([
      nav.userAgent || "",
      nav.language || "",
      (nav.languages || []).join(","),
      nav.platform || "",
      nav.hardwareConcurrency || "",
      nav.maxTouchPoints || "",
      screenInfo.width || "",
      screenInfo.height || "",
      screenInfo.colorDepth || "",
      window.devicePixelRatio || "",
      new Date().getTimezoneOffset(),
      canvasSignal(),
      webglSignal(),
      fontSignal()
    ].join("||"));
  }

  var browserId = readCookie(BID_COOKIE);
  if (!/^[a-f0-9]{32}$/.test(browserId)) {
    browserId = randomHex(16);
  }
  var fingerprint = buildFingerprint();
  setCookie(BID_COOKIE, browserId, BID_MAX_AGE);
  setCookie(FP_COOKIE, fingerprint, FP_MAX_AGE);
  window.__cdnGuardFingerprint = { id: browserId, fingerprint: fingerprint };
})();
