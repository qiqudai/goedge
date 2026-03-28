(function () {
  function getCookie(name) {
    var value = "; " + document.cookie;
    var parts = value.split("; " + name + "=");
    if (parts.length === 2) {
      return parts.pop().split(";").shift();
    }
  }

  function setRet(guard, payload) {
    var keyPart = guard.substr(0, 8);
    var key = cdn.MD5(keyPart);
    var enc = cdn.centos.encrypt(JSON.stringify(payload), key, { iv: key });
    document.cookie = "guardret=" + enc.toString();
    window.location.reload();
  }

  var img = document.getElementById("img");
  var degInput = document.getElementById("deg");
  var degText = document.getElementById("degText");
  var access = document.getElementById("access");
  var refresh = document.getElementById("refresh");

  function sync() {
    var val = parseInt((degInput && degInput.value) || "0", 10);
    if (isNaN(val)) val = 0;
    if (degText) degText.innerText = "" + val;
    if (img) img.style.transform = "rotate(" + val + "deg)";
  }

  if (degInput) {
    degInput.addEventListener("input", sync);
    degInput.addEventListener("change", sync);
  }
  sync();

  if (access) {
    access.addEventListener("click", function (e) {
      e.preventDefault();
      var guard = getCookie("guard");
      if (!guard) {
        window.location.reload();
        return;
      }
      var val = parseInt((degInput && degInput.value) || "0", 10);
      if (isNaN(val)) val = 0;
      setRet(guard, { deg: val });
    });
  }

  if (refresh && img) {
    refresh.addEventListener("click", function (e) {
      e.preventDefault();
      img.src = "/_guard/rotate_image?r=" + Math.random();
      if (degInput) degInput.value = "0";
      sync();
    });
  }
})();

