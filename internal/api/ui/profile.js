const UI = StatocystUI;

function setStatus(message, warn = false) {
  const el = UI.$("profileStatus");
  if (!warn && message === "Profile loaded.") {
    el.style.display = "none";
    return;
  }
  el.style.display = "block";
  el.textContent = message;
  el.className = warn ? "status warn" : "status";
}

function formatDate(raw) {
  if (!raw) return "-";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  return d.toLocaleString();
}

function daysAgoLabel(raw) {
  if (!raw) return "Joined recently";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return "Joined recently";
  const msPerDay = 24 * 60 * 60 * 1000;
  const days = Math.max(0, Math.floor((Date.now() - d.getTime()) / msPerDay));
  const unit = days === 1 ? "day" : "days";
  return `Joined ${days} ${unit} ago`;
}

function renderOrgs(memberships) {
  const target = UI.$("profileOrgs");
  target.innerHTML = "";

  function renderEmpty() {
    target.append("No organizations yet. ");
    const link = document.createElement("a");
    link.href = "/organization";
    link.textContent = "Create one on /organization";
    target.append(link);
    target.append(".");
  }

  if (!Array.isArray(memberships) || memberships.length === 0) {
    renderEmpty();
    return;
  }

  const list = memberships
    .map((m) => m?.org?.name)
    .filter((name) => typeof name === "string" && name.trim() !== "");

  if (list.length === 0) {
    renderEmpty();
    return;
  }

  const ul = document.createElement("ul");
  ul.className = "list";
  for (const name of list) {
    const li = document.createElement("li");
    li.textContent = name;
    ul.appendChild(li);
  }
  target.appendChild(ul);
}

function renderProfile(me, orgs) {
  const human = me?.data?.human;
  UI.$("profileEmail").textContent = human?.email || "-";
  UI.$("profileJoined").textContent = daysAgoLabel(human?.created_at);
  renderOrgs(orgs?.data?.memberships);

  const isSuperAdmin = Boolean(me?.data?.is_super_admin);
  UI.$("superAdminRow").style.display = isSuperAdmin ? "block" : "none";
}

async function init() {
  UI.initTopNav();
  setStatus("Loading profile...");

  const [me, orgs] = await Promise.all([UI.req("/v1/me"), UI.req("/v1/me/orgs")]);

  if (me.status !== 200) {
    setStatus("Could not load profile. Please login again.", true);
    return;
  }
  if (orgs.status !== 200) {
    setStatus("Profile loaded, but organizations could not be loaded.", true);
    renderProfile(me, { data: { memberships: [] } });
    return;
  }

  renderProfile(me, orgs);
  setStatus("Profile loaded.");
}

init().catch((err) => {
  setStatus(`Unexpected error: ${String(err)}`, true);
});
