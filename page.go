package main

const loginPageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Antigravity Daily 登录</title>
  <script>
    (() => {
      const resolve = () => {
        let selected = 'auto';
        try {
          const saved = JSON.parse(localStorage.getItem('cli-proxy-theme') || '{}');
          selected = saved?.state?.theme || saved?.theme || 'auto';
        } catch (_) {}
        if (selected === 'auto') return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        return selected === 'white' ? 'white' : selected === 'dark' ? 'dark' : 'light';
      };
      window.__applyCPATheme = () => document.documentElement.setAttribute('data-theme', resolve());
      window.__applyCPATheme();
    })();
  </script>
  <style>
    :root{color-scheme:light;font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;--bg-secondary:#f7f5ef;--bg-primary:#f0eee8;--bg-tertiary:#e9e6df;--text-primary:#2d2a26;--text-secondary:#6d6760;--border-color:#d5d2cb;--primary:#8b8680;--primary-hover:#7f7a74;--input-bg:#fffdf9;--status-bg:#e9e6df;--shadow:0 16px 48px #3b352515}
    [data-theme=white]{color-scheme:light;--bg-secondary:#fff;--bg-primary:#fff;--bg-tertiary:#f6f6f6;--text-primary:#2d2a26;--text-secondary:#6d6760;--border-color:#d9d9d9;--primary:#8b8680;--primary-hover:#7f7a74;--input-bg:#fff;--status-bg:#f6f6f6;--shadow:0 16px 48px #00000012}
    [data-theme=dark]{color-scheme:dark;--bg-secondary:#151412;--bg-primary:#1d1b18;--bg-tertiary:#272521;--text-primary:#f2eee8;--text-secondary:#b5aea4;--border-color:#3a3732;--primary:#9d9891;--primary-hover:#b0aaa2;--input-bg:#151412;--status-bg:#272521;--shadow:0 16px 48px #0007}
    body{margin:0;background:var(--bg-secondary);color:var(--text-primary);transition:background-color .18s,color .18s}.wrap{max-width:760px;margin:40px auto;padding:0 20px}
    .card{background:var(--bg-primary);border:1px solid var(--border-color);border-radius:16px;padding:24px;box-shadow:var(--shadow)}
    h1{margin:0 0 10px;font-size:25px}.hint{color:var(--text-secondary);line-height:1.6}.row{margin:18px 0}
    label{display:block;margin-bottom:8px;font-weight:650}input,textarea,button{box-sizing:border-box;width:100%;border-radius:10px;border:1px solid #394568;font:inherit}
    input,textarea{padding:11px 12px;background:var(--input-bg);color:var(--text-primary);border-color:var(--border-color)}textarea{min-height:100px;resize:vertical}
    button{padding:12px 16px;background:var(--primary);color:white;border:0;font-weight:750;cursor:pointer}button:hover{background:var(--primary-hover)}button.secondary{background:var(--bg-tertiary);color:var(--text-primary);border:1px solid var(--border-color)}button:disabled{opacity:.55;cursor:wait}
    .status{margin-top:18px;padding:12px;border-radius:10px;background:var(--status-bg);white-space:pre-wrap;min-height:24px}.ok{color:#10b981}.err{color:#c65746}
    code{background:var(--bg-tertiary);padding:.15rem .35rem;border-radius:5px}.steps{line-height:1.65;padding-left:22px}
  </style>
</head>
<body><main class="wrap"><section class="card">
  <h1>Antigravity Daily 凭证生成器</h1>
  <p class="hint">该插件通过 GCLI2API 使用的 daily endpoint 获取真实 <code>project_id</code>，完成后由 CPA Host 保存凭证。管理密钥只保存在当前页面内存中。</p>
  <ol class="steps"><li>输入 CPA 管理密钥并启动登录。</li><li>在新窗口完成 Google 授权；浏览器最后可能显示 localhost 无法连接，这是正常的。</li><li>复制浏览器地址栏中的完整回调 URL，粘贴到下方并提交。</li></ol>
  <div class="row"><label for="key">CPA 管理密钥</label><input id="key" type="password" autocomplete="off" placeholder="Management Key"></div>
  <div class="row"><button id="start">1. 启动 Google 登录</button></div>
  <div class="row"><label for="callback">Google 授权后的完整回调 URL</label><textarea id="callback" spellcheck="false" placeholder="http://localhost:51121/oauth-callback?state=...&code=..."></textarea></div>
  <div class="row"><button id="submit" class="secondary" disabled>2. 提交回调并生成凭证</button></div>
  <div id="status" class="status">等待开始。</div>
</section></main>
<script>
(() => {
  const key = document.querySelector('#key');
  const callback = document.querySelector('#callback');
  const start = document.querySelector('#start');
  const submit = document.querySelector('#submit');
  const status = document.querySelector('#status');
  let state = '';
  addEventListener('storage', event => { if (event.key === 'cli-proxy-theme') window.__applyCPATheme?.(); });
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => window.__applyCPATheme?.());
  const headers = () => ({'X-Management-Key': key.value.trim()});
  const show = (text, kind='') => { status.textContent = text; status.className = 'status ' + kind; };
  const json = async response => { const body = await response.json().catch(() => ({})); if (!response.ok) throw new Error(body.error || ('HTTP ' + response.status)); return body; };
  start.addEventListener('click', async () => {
    if (!key.value.trim()) return show('请先输入 CPA 管理密钥。', 'err');
    start.disabled = true; submit.disabled = true; show('正在生成 Google 授权链接…');
    try {
      const data = await fetch('/v0/management/antigravity-daily-auth-url', {headers: headers()}).then(json);
      state = data.state;
      if (!state || !data.url) throw new Error('CPA 没有返回登录 URL/state');
      window.open(data.url, '_blank', 'noopener,noreferrer');
      submit.disabled = false;
      show('登录窗口已打开。完成授权后复制地址栏中的完整 localhost 回调 URL。');
    } catch (error) { show('启动失败：' + error.message, 'err'); }
    finally { start.disabled = false; }
  });
  submit.addEventListener('click', async () => {
    if (!state) return show('请先启动登录。', 'err');
    if (!callback.value.trim()) return show('请粘贴完整回调 URL。', 'err');
    submit.disabled = true; show('正在提交回调并通过 daily endpoint 获取 project_id…');
    try {
      await fetch('/v0/management/oauth-callback', {method:'POST', headers:{...headers(),'Content-Type':'application/json'}, body:JSON.stringify({provider:'antigravity-daily', redirect_url:callback.value.trim()})}).then(json);
      for (let i=0; i<40; i++) {
        const result = await fetch('/v0/management/get-auth-status?state=' + encodeURIComponent(state), {headers:headers()}).then(json);
        if (result.status === 'ok') { show('成功：CPA 已保存带 project_id 的 Antigravity 凭证。', 'ok'); callback.value=''; state=''; return; }
        if (result.status === 'error') throw new Error(result.error || '认证失败');
        show('正在处理：OAuth token → 用户邮箱 → daily loadCodeAssist/onboardUser…');
        await new Promise(resolve => setTimeout(resolve, 1500));
      }
      throw new Error('等待 CPA 保存凭证超时');
    } catch (error) { show('生成失败：' + error.message, 'err'); }
    finally { submit.disabled = false; }
  });
})();
</script></body></html>`
