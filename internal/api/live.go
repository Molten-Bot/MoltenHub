package api

import (
	_ "embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"statocyst/internal/model"
)

//go:embed ui/live.html
var uiLiveHTML []byte

type livePageData struct {
	AppName       string
	GeneratedAt   string
	Organizations []liveOrgView
}

type liveOrgView struct {
	Handle string
	Humans []liveHumanView
	Agents []liveAgentView
}

type liveHumanView struct {
	Handle string
	Agents []liveAgentView
}

type liveAgentView struct {
	Handle string
}

func (h *Handler) handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	tplSrc := uiLiveHTML
	if uiDevMode {
		path := filepath.Clean(filepath.Join("internal", "api", "ui", "live.html"))
		if fromDisk, err := os.ReadFile(path); err == nil {
			tplSrc = fromDisk
		}
		w.Header().Set("Cache-Control", "no-store")
	}

	tpl, err := template.New("live").Parse(string(tplSrc))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "template_error", "failed to render live page")
		return
	}
	data := h.buildLivePageData(h.now().UTC())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = tpl.Execute(w, data)
}

func (h *Handler) buildLivePageData(now time.Time) livePageData {
	snapshot := h.store.AdminSnapshot()

	publicOrgs := make(map[string]model.Organization)
	for _, org := range snapshot.Organizations {
		if org.IsPublic {
			publicOrgs[org.OrgID] = org
		}
	}
	publicHumans := make(map[string]model.Human)
	for _, human := range snapshot.Humans {
		if human.IsPublic {
			publicHumans[human.HumanID] = human
		}
	}

	humanByOrg := make(map[string][]model.Human)
	for _, membership := range snapshot.Memberships {
		if membership.Status != model.StatusActive {
			continue
		}
		if _, ok := publicOrgs[membership.OrgID]; !ok {
			continue
		}
		human, ok := publicHumans[membership.HumanID]
		if !ok {
			continue
		}
		humanByOrg[membership.OrgID] = append(humanByOrg[membership.OrgID], human)
	}

	agentsByHuman := make(map[string][]model.Agent)
	orgAgents := make(map[string][]model.Agent)
	for _, agent := range snapshot.Agents {
		if agent.Status != model.StatusActive || !agent.IsPublic {
			continue
		}
		if _, ok := publicOrgs[agent.OrgID]; !ok {
			continue
		}
		if agent.OwnerHumanID != nil {
			if _, ok := publicHumans[*agent.OwnerHumanID]; !ok {
				continue
			}
			key := agent.OrgID + "\x00" + *agent.OwnerHumanID
			agentsByHuman[key] = append(agentsByHuman[key], agent)
			continue
		}
		orgAgents[agent.OrgID] = append(orgAgents[agent.OrgID], agent)
	}

	orgIDs := make([]string, 0, len(publicOrgs))
	for orgID := range publicOrgs {
		orgIDs = append(orgIDs, orgID)
	}
	sort.Slice(orgIDs, func(i, j int) bool {
		return publicOrgs[orgIDs[i]].Name < publicOrgs[orgIDs[j]].Name
	})

	orgViews := make([]liveOrgView, 0, len(orgIDs))
	for _, orgID := range orgIDs {
		humans := dedupeHumansByID(humanByOrg[orgID])
		sort.Slice(humans, func(i, j int) bool {
			return humans[i].Handle < humans[j].Handle
		})

		humanViews := make([]liveHumanView, 0, len(humans))
		for _, human := range humans {
			key := orgID + "\x00" + human.HumanID
			owned := agentsByHuman[key]
			sort.Slice(owned, func(i, j int) bool {
				return owned[i].AgentID < owned[j].AgentID
			})
			agentViews := make([]liveAgentView, 0, len(owned))
			for _, agent := range owned {
				agentViews = append(agentViews, liveAgentView{Handle: agent.AgentID})
			}
			humanViews = append(humanViews, liveHumanView{
				Handle: human.Handle,
				Agents: agentViews,
			})
		}

		ownedByOrg := orgAgents[orgID]
		sort.Slice(ownedByOrg, func(i, j int) bool {
			return ownedByOrg[i].AgentID < ownedByOrg[j].AgentID
		})
		orgAgentViews := make([]liveAgentView, 0, len(ownedByOrg))
		for _, agent := range ownedByOrg {
			orgAgentViews = append(orgAgentViews, liveAgentView{Handle: agent.AgentID})
		}

		orgViews = append(orgViews, liveOrgView{
			Handle: publicOrgs[orgID].Name,
			Humans: humanViews,
			Agents: orgAgentViews,
		})
	}

	return livePageData{
		AppName:       uiAppName(),
		GeneratedAt:   now.Format(time.RFC3339),
		Organizations: orgViews,
	}
}

func dedupeHumansByID(items []model.Human) []model.Human {
	if len(items) == 0 {
		return nil
	}
	out := make([]model.Human, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, human := range items {
		if human.HumanID == "" {
			continue
		}
		if _, ok := seen[human.HumanID]; ok {
			continue
		}
		seen[human.HumanID] = struct{}{}
		out = append(out, human)
	}
	return out
}
