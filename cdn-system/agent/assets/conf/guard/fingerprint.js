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

  // Use only stable browser signals. Safari randomizes canvas/webgl fingerprints.
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
      new Date().getTimezoneOffset()
    ].join("||"));
  }

  var browserId = readCookie(BID_COOKIE);
  if (!/^[a-f0-9]{32}$/.test(browserId)) {
    browserId = randomHex(16);
  }
  var fingerprint = buildFingerprint() || hash(browserId);
  setCookie(BID_COOKIE, browserId, BID_MAX_AGE);
  setCookie(FP_COOKIE, fingerprint, FP_MAX_AGE);
  window.__cdnGuardFingerprint = { id: browserId, fingerprint: fingerprint };
})();
