const root = document.querySelector("#app");
const toastNode = document.querySelector("#toast");
let state = { tab: "accounts", accounts: [], agents: [], users: [] };

const escapeHTML = (value = "") => String(value)
  .replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;");

async function api(path, options = {}) {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
  });
  const content = response.headers.get("content-type")?.includes("json")
    ? await response.json() : await response.text();
  if (!response.ok) throw new Error(content?.error || content || `HTTP ${response.status}`);
  return content;
}

function toast(message) {
  toastNode.textContent = message;
  toastNode.classList.add("show");
  setTimeout(() => toastNode.classList.remove("show"), 2400);
}

async function boot() {
  const status = await api("/api/admin/status");
  if (!status.authenticated) return renderAuth(status.setup_required);
  await load();
}

function renderAuth(setup) {
  root.innerHTML = `<div class="center"><div class="auth">
    <div class="brand"><div class="logo">爪</div><div><h1>WeClaw</h1><div class="muted">微信 Agent 管理</div></div></div>
    <form class="card stack" id="auth-form">
      <div><h2>${setup ? "创建管理员" : "管理员登录"}</h2>
      <p class="muted">${setup ? "密码仅保存在当前 WeClaw 数据目录。" : "管理微信账号、Agent 和成员权限。"}</p></div>
      <label>密码<input type="password" name="password" minlength="10" required autofocus></label>
      <button class="btn primary">${setup ? "完成初始化" : "登录"}</button>
    </form></div></div>`;
  document.querySelector("#auth-form").onsubmit = async (event) => {
    event.preventDefault();
    const password = new FormData(event.target).get("password");
    try {
      await api(setup ? "/api/admin/setup" : "/api/admin/login", {
        method: "POST", body: JSON.stringify({ password }),
      });
      await load();
    } catch (error) { toast(error.message); }
  };
}

async function load() {
  [state.accounts, state.agents, state.users] = await Promise.all([
    api("/api/admin/accounts"), api("/api/admin/agents"), api("/api/admin/users"),
  ]);
  render();
}

function render() {
  root.innerHTML = `<div class="app">
    <header><div class="brand"><div class="logo">爪</div><h1>WeClaw</h1></div>
      <button class="btn small" id="logout">退出登录</button></header>
    <main><nav class="tabs">
      ${tab("accounts", "微信账号")}${tab("agents", "Agent")}${tab("users", "用户权限")}
    </nav><section id="page"></section></main></div><div id="modal"></div>`;
  document.querySelector("#logout").onclick = async () => {
    await api("/api/admin/logout", { method: "POST" }); boot();
  };
  document.querySelectorAll(".tab").forEach((button) => button.onclick = () => {
    state.tab = button.dataset.tab; render();
  });
  ({ accounts: renderAccounts, agents: renderAgents, users: renderUsers })[state.tab]();
}

function tab(id, label) {
  return `<button class="tab ${state.tab === id ? "active" : ""}" data-tab="${id}">${label}</button>`;
}

function pageHeader(eyebrow, title, text, action = "") {
  return `<div class="page-head"><div><div class="eyebrow">${eyebrow}</div>
    <h2>${title}</h2><div class="muted">${text}</div></div>${action}</div>`;
}

function renderAccounts() {
  const cards = state.accounts.map((item) => `<article class="card item">
    <div class="item-head"><div><h3>微信账号</h3><div class="meta">${escapeHTML(item.bot_id)}</div></div>
    <span class="badge ${item.online ? "good" : "bad"}"><i class="dot"></i>${item.online ? "运行中" : "已停止"}</span></div>
    <div class="actions"><button class="btn small danger account-delete" data-id="${escapeHTML(item.bot_id)}">删除账号</button></div>
  </article>`).join("");
  document.querySelector("#page").innerHTML =
    pageHeader("ACCOUNTS", "微信账号", "在浏览器中扫码添加账号，无需进入容器。", `<button class="btn primary" id="add-account">添加微信账号</button>`) +
    `<div class="grid">${cards || `<div class="card empty">尚未登录微信账号</div>`}</div>`;
  document.querySelector("#add-account").onclick = startLogin;
  document.querySelectorAll(".account-delete").forEach((button) => button.onclick = async () => {
    if (!confirm("删除该微信登录账号？")) return;
    await api(`/api/admin/accounts?bot_id=${encodeURIComponent(button.dataset.id)}`, { method: "DELETE" });
    toast("账号已删除"); await load();
  });
}

async function startLogin() {
  try {
    const session = await api("/api/admin/login-sessions", { method: "POST", body: "{}" });
    showModal(`<div class="modal-head"><div><h2>扫码登录微信</h2><div class="muted">使用微信扫描并在手机上确认</div></div><button class="btn small modal-close">关闭</button></div>
      <div class="modal-body"><img class="qr" src="/api/admin/login-sessions/${session.id}/qr.png">
      <div id="qr-status" class="muted" style="text-align:center">等待扫码</div></div>`);
    pollLogin(session.id);
  } catch (error) { toast(error.message); }
}

async function pollLogin(id) {
  const status = await api(`/api/admin/login-sessions/${id}`);
  const labels = { waiting: "等待扫码", scanned: "已扫码，请在手机上确认", connected: "登录完成", expired: "二维码已过期", failed: status.error || "登录失败" };
  const node = document.querySelector("#qr-status");
  if (node) node.textContent = labels[status.status] || status.status;
  if (status.status === "connected") {
    toast("微信账号已连接"); setTimeout(load, 700); return;
  }
  if (!["expired", "failed"].includes(status.status) && node) setTimeout(() => pollLogin(id), 1200);
}

function renderAgents() {
  const cards = state.agents.map((agent) => `<article class="card item">
    <div class="item-head"><div><h3>${escapeHTML(agent.name)}</h3><div class="meta">${escapeHTML(agent.type)} · ${escapeHTML(agent.endpoint || agent.command || "未配置地址")}</div></div>
    <span class="badge ${agent.public ? "good" : "warn"}">${agent.public ? "所有用户" : "指定用户"}</span></div>
    <p class="muted" style="margin-top:18px">${agent.default ? "全局默认 Agent" : `别名：${escapeHTML((agent.aliases || []).join(", ") || "无")}`}</p>
    <div class="actions"><button class="btn small agent-edit" data-name="${escapeHTML(agent.name)}">编辑</button>
    <button class="btn small agent-public" data-name="${escapeHTML(agent.name)}">${agent.public ? "改为指定用户" : "允许所有用户"}</button></div>
  </article>`).join("");
  document.querySelector("#page").innerHTML =
    pageHeader("AGENTS", "Agent", "统一管理 Native、HTTP、ACP 和 CLI Agent。", `<button class="btn primary" id="add-agent">添加 Agent</button>`) +
    `<div class="grid">${cards || `<div class="card empty">尚未配置 Agent</div>`}</div>`;
  document.querySelector("#add-agent").onclick = () => editAgent(null);
  document.querySelectorAll(".agent-edit").forEach((button) => button.onclick = () =>
    editAgent(state.agents.find((item) => item.name === button.dataset.name)));
  document.querySelectorAll(".agent-public").forEach((button) => button.onclick = async () => {
    const agent = state.agents.find((item) => item.name === button.dataset.name);
    await api(`/api/admin/agents/${encodeURIComponent(agent.name)}/public`, {
      method: "POST", body: JSON.stringify({ public: !agent.public }),
    });
    toast("访问范围已更新"); await load();
  });
}

function editAgent(agent) {
  const value = agent || { type: "native", aliases: [], public: false };
  showModal(`<div class="modal-head"><div><h2>${agent ? "编辑" : "添加"} Agent</h2><div class="muted">密钥留空时保留已有值</div></div>
    <button class="btn small modal-close">关闭</button></div><form class="modal-body form-grid" id="agent-form">
    <label>名称<input name="name" value="${escapeHTML(value.name)}" ${agent ? "readonly" : ""} required pattern="[a-zA-Z0-9_-]+"></label>
    <label>类型<select name="type">${["native","http","acp","cli"].map((type) => `<option ${value.type === type ? "selected" : ""}>${type}</option>`).join("")}</select></label>
    <label class="wide">Endpoint（Native / HTTP）<input name="endpoint" value="${escapeHTML(value.endpoint)}"></label>
    <label class="wide">Command（ACP / CLI）<input name="command" value="${escapeHTML(value.command)}"></label>
    <label>模型<input name="model" value="${escapeHTML(value.model)}"></label>
    <label>别名，逗号分隔<input name="aliases" value="${escapeHTML((value.aliases || []).join(","))}"></label>
    <label>API Key<input type="password" name="api_key"></label>
    <label>主动消息令牌<input type="password" name="outbound_token"></label>
    <label><span>请求超时（秒）</span><input type="number" name="timeout_seconds" value="${value.timeout_seconds || 180}"></label>
    <label><span>全局默认</span><select name="default"><option value="false">否</option><option value="true" ${value.default ? "selected" : ""}>是</option></select></label>
    <label class="wide"><span>访问范围</span><select name="public"><option value="false">由用户权限决定</option><option value="true" ${value.public ? "selected" : ""}>所有用户</option></select></label>
    <div class="actions wide"><button class="btn primary">保存 Agent</button>${agent && !agent.default ? `<button type="button" class="btn danger" id="delete-agent">删除</button>` : ""}</div>
  </form>`);
  document.querySelector("#agent-form").onsubmit = async (event) => {
    event.preventDefault();
    const data = Object.fromEntries(new FormData(event.target));
    data.aliases = data.aliases.split(",").map((item) => item.trim()).filter(Boolean);
    data.timeout_seconds = Number(data.timeout_seconds);
    data.public = data.public === "true"; data.default = data.default === "true";
    await api(`/api/admin/agents/${encodeURIComponent(data.name)}`, { method: "PUT", body: JSON.stringify(data) });
    closeModal(); toast("Agent 已保存"); await load();
  };
  const remove = document.querySelector("#delete-agent");
  if (remove) remove.onclick = async () => {
    if (!confirm("删除该 Agent？")) return;
    await api(`/api/admin/agents/${encodeURIComponent(agent.name)}`, { method: "DELETE" });
    closeModal(); toast("Agent 已删除"); await load();
  };
}

function renderUsers() {
  const cards = state.users.map((user) => `<article class="card item">
    <div class="item-head"><div><h3>${escapeHTML(user.user_id)}</h3><div class="meta">账号 ${escapeHTML(user.account_id)}</div></div>
    <span class="badge ${user.status === "active" ? "good" : user.status === "blocked" ? "bad" : "warn"}">${user.status}</span></div>
    <div class="permissions">${state.agents.map((agent) => `<label class="check"><input type="checkbox" class="permission"
      data-account="${escapeHTML(user.account_id)}" data-user="${escapeHTML(user.user_id)}" data-agent="${escapeHTML(agent.name)}"
      ${user.permissions?.[agent.name] ? "checked" : ""} ${agent.public ? "disabled" : ""}>${escapeHTML(agent.name)}${agent.public ? "（公开）" : ""}</label>`).join("")}</div>
    <div class="actions"><button class="btn small user-edit" data-key="${escapeHTML(user.account_id + "|" + user.user_id)}">状态与默认 Agent</button></div>
  </article>`).join("");
  document.querySelector("#page").innerHTML =
    pageHeader("ACCESS", "用户权限", "联系人首次发送消息后自动出现在这里。") +
    `<div class="grid">${cards || `<div class="card empty">尚未发现微信联系人</div>`}</div>`;
  document.querySelectorAll(".permission").forEach((input) => input.onchange = async () => {
    await api(`/api/admin/users/${encodeURIComponent(input.dataset.account)}/${encodeURIComponent(input.dataset.user)}/permissions`, {
      method: "PUT", body: JSON.stringify({ agent: input.dataset.agent, allowed: input.checked }),
    });
    toast("权限已更新");
  });
  document.querySelectorAll(".user-edit").forEach((button) => button.onclick = () => {
    const [account, userID] = button.dataset.key.split("|");
    editUser(state.users.find((item) => item.account_id === account && item.user_id === userID));
  });
}

function editUser(user) {
  const allowed = state.agents.filter((agent) => user.permissions?.[agent.name]);
  showModal(`<div class="modal-head"><div><h2>用户设置</h2><div class="meta">${escapeHTML(user.user_id)}</div></div><button class="btn small modal-close">关闭</button></div>
    <form class="modal-body stack" id="user-form"><label>状态<select name="status">${["pending","active","blocked"].map((value) => `<option ${user.status === value ? "selected" : ""}>${value}</option>`).join("")}</select></label>
    <label>默认 Agent<select name="default_agent"><option value="">使用全局默认</option>${allowed.map((agent) => `<option value="${escapeHTML(agent.name)}" ${user.default_agent === agent.name ? "selected" : ""}>${escapeHTML(agent.name)}</option>`).join("")}</select></label>
    <button class="btn primary">保存用户</button></form>`);
  document.querySelector("#user-form").onsubmit = async (event) => {
    event.preventDefault();
    const data = Object.fromEntries(new FormData(event.target));
    await api(`/api/admin/users/${encodeURIComponent(user.account_id)}/${encodeURIComponent(user.user_id)}`, { method: "PUT", body: JSON.stringify(data) });
    closeModal(); toast("用户设置已保存"); await load();
  };
}

function showModal(content) {
  document.querySelector("#modal").innerHTML = `<div class="modal-bg"><div class="modal">${content}</div></div>`;
  document.querySelectorAll(".modal-close").forEach((button) => button.onclick = closeModal);
  document.querySelector(".modal-bg").onclick = (event) => { if (event.target.classList.contains("modal-bg")) closeModal(); };
}
function closeModal() { document.querySelector("#modal").innerHTML = ""; }
boot().catch((error) => { root.innerHTML = `<div class="center"><div class="card">${escapeHTML(error.message)}</div></div>`; });
