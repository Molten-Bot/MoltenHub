const $ = (id) => document.getElementById(id);

const headers = () => {
  const h = { "Content-Type": "application/json" };
  const token = $("authToken").value.trim();
  if (token) h.Authorization = `Bearer ${token}`;
  const humanId = $("humanId").value.trim();
  const humanEmail = $("humanEmail").value.trim();
  if (humanId) h["X-Human-Id"] = humanId;
  if (humanEmail) h["X-Human-Email"] = humanEmail;
  return h;
};

const selectedOrg = () => $("orgSelect").value.trim();

async function req(path, method = "GET", body = null) {
  const res = await fetch(path, {
    method,
    headers: headers(),
    body: body ? JSON.stringify(body) : null,
  });
  const text = await res.text();
  let data = text;
  try {
    data = JSON.parse(text || "{}");
  } catch (_) {}
  return { status: res.status, data };
}

function out(el, obj) {
  $(el).textContent = JSON.stringify(obj, null, 2);
}

async function listOrgs() {
  const r = await req("/v1/me/orgs");
  out("orgOut", r);
  const select = $("orgSelect");
  select.innerHTML = "";
  if (r.status === 200 && Array.isArray(r.data.memberships)) {
    for (const m of r.data.memberships) {
      const opt = document.createElement("option");
      opt.value = m.org.org_id;
      opt.textContent = `${m.org.name} (${m.membership.role})`;
      select.appendChild(opt);
    }
  }
}

$("btnMe").onclick = async () => out("meOut", await req("/v1/me"));
$("btnCreateOrg").onclick = async () =>
  out("orgOut", await req("/v1/orgs", "POST", { name: $("orgName").value }));
$("btnListOrgs").onclick = listOrgs;

$("btnInvite").onclick = async () =>
  out(
    "inviteOut",
    await req(`/v1/orgs/${selectedOrg()}/invites`, "POST", {
      email: $("inviteEmail").value,
      role: $("inviteRole").value,
    })
  );
$("btnAcceptInvite").onclick = async () =>
  out("inviteOut", await req(`/v1/org-invites/${$("inviteId").value}/accept`, "POST"));
$("btnOrgHumans").onclick = async () =>
  out("inviteOut", await req(`/v1/orgs/${selectedOrg()}/humans`));

$("btnRegisterAgent").onclick = async () => {
  const owner = $("ownerHumanId").value.trim();
  const payload = { org_id: selectedOrg(), agent_id: $("agentId").value };
  if (owner) payload.owner_human_id = owner;
  out("agentOut", await req("/v1/agents/register", "POST", payload));
};
$("btnRotateAgent").onclick = async () =>
  out("agentOut", await req(`/v1/agents/${$("rotateAgentId").value}/rotate-token`, "POST"));
$("btnRevokeAgent").onclick = async () =>
  out("agentOut", await req(`/v1/agents/${$("rotateAgentId").value}`, "DELETE"));
$("btnListAgents").onclick = async () =>
  out("agentOut", await req(`/v1/orgs/${selectedOrg()}/agents`));

$("btnReqOrgTrust").onclick = async () =>
  out(
    "orgTrustOut",
    await req("/v1/org-trusts", "POST", {
      org_id: selectedOrg(),
      peer_org_id: $("peerOrgId").value,
    })
  );
$("btnApproveOrgTrust").onclick = async () =>
  out("orgTrustOut", await req(`/v1/org-trusts/${$("orgTrustId").value}/approve`, "POST"));
$("btnBlockOrgTrust").onclick = async () =>
  out("orgTrustOut", await req(`/v1/org-trusts/${$("orgTrustId").value}/block`, "POST"));
$("btnRevokeOrgTrust").onclick = async () =>
  out("orgTrustOut", await req(`/v1/org-trusts/${$("orgTrustId").value}`, "DELETE"));

$("btnReqAgentTrust").onclick = async () =>
  out(
    "agentTrustOut",
    await req("/v1/agent-trusts", "POST", {
      org_id: selectedOrg(),
      agent_id: $("trustAgentId").value,
      peer_agent_id: $("trustPeerAgentId").value,
    })
  );
$("btnApproveAgentTrust").onclick = async () =>
  out("agentTrustOut", await req(`/v1/agent-trusts/${$("agentTrustId").value}/approve`, "POST"));
$("btnBlockAgentTrust").onclick = async () =>
  out("agentTrustOut", await req(`/v1/agent-trusts/${$("agentTrustId").value}/block`, "POST"));
$("btnRevokeAgentTrust").onclick = async () =>
  out("agentTrustOut", await req(`/v1/agent-trusts/${$("agentTrustId").value}`, "DELETE"));

$("btnGraph").onclick = async () =>
  out("graphOut", await req(`/v1/orgs/${selectedOrg()}/trust-graph`));
$("btnAudit").onclick = async () =>
  out("graphOut", await req(`/v1/orgs/${selectedOrg()}/audit`));
$("btnStats").onclick = async () =>
  out("graphOut", await req(`/v1/orgs/${selectedOrg()}/stats`));

listOrgs();
