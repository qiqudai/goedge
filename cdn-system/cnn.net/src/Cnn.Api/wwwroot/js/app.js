window.cnn = window.cnn || {};
window.cnn.storage = {
  get: function (key) {
    try {
      return window.localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set: function (key, value) {
    try {
      window.localStorage.setItem(key, value ?? '');
    } catch {
      // ignore
    }
  },
  remove: function (key) {
    try {
      window.localStorage.removeItem(key);
    } catch {
      // ignore
    }
  }
};
window.cnn.theme = {
  set: function (theme) {
    if (!theme) return;
    document.documentElement.setAttribute('data-theme', theme);
  }
};

window.cnn.downloadText = function (filename, text, contentType) {
  try {
    var blob = new Blob([text ?? ''], { type: contentType || 'text/plain;charset=utf-8' });
    var url = window.URL.createObjectURL(blob);
    var link = document.createElement('a');
    link.href = url;
    link.download = filename || 'download.txt';
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  } catch {
    // ignore
  }
};
