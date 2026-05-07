function copyText(text) {
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text);
  } else {
    const el = document.createElement('textarea');
    el.value = text;
    el.setAttribute('readonly', ''); // 隐藏输入法
    el.style.position = 'absolute';
    el.style.left = '-9999px';      // 移出屏幕外
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  }
}

window.dnotifyTimer = null; window.dnotifyEl = document.body.appendChild(Object.assign(document.createElement('div'), { 
  id: 'dnotify', style: `display:none; min-width: 260px;max-width: 40%; padding: 10px; box-sizing: border-box; border: 1px solid #ebeef5; text-align: center; color:#333;
    position: fixed; background-color: #fff; top:16px;right:16px;z-index: 9999999; font-size: 14px;line-height: 1.4; border-radius: 8px; box-shadow: 0 2px 12px 0 rgba(0,0,0,.1); `
}));
function notify(txt, type, time) {
  dnotifyEl.style.display = 'block';
  dnotifyEl.style.color = ['#333', '#67C23A', '#F56C6C', '#E6A23C'][type] // 0 info, 1 success, 2 error, 3 warning
  dnotifyEl.style.backgroundColor = ['#fff', '#f0f9eb', '#fef0f0', '#fdf6ec'][type]
  dnotifyEl.innerHTML = txt;
  clearTimeout(dnotifyTimer);
  dnotifyTimer = setTimeout(() => dnotifyEl.style.display = 'none', (time || 3) * 1000);
}