package api

import (
	"sort"
	"strings"

	"moltenhub/internal/model"
)

type agentSkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type agentPeerSkillSummary struct {
	AgentUUID string              `json:"agent_uuid,omitempty"`
	AgentID   string              `json:"agent_id,omitempty"`
	AgentURI  string              `json:"agent_uri"`
	Skills    []agentSkillSummary `json:"skills"`
}

func parseAdvertisedSkills(metadata map[string]any) []agentSkillSummary {
	raw, ok := metadata[model.AgentMetadataKeySkills]
	if !ok {
		return []agentSkillSummary{}
	}
	items := []map[string]any{}
	switch typed := raw.(type) {
	case []any:
		for _, entry := range typed {
			obj, ok := entry.(map[string]any)
			if ok {
				items = append(items, obj)
			}
		}
	case []map[string]any:
		items = append(items, typed...)
	default:
		return []agentSkillSummary{}
	}

	skillsByName := map[string]agentSkillSummary{}
	for _, item := range items {
		rawName, _ := item["name"].(string)
		name, valid := normalizeSkillName(rawName)
		if !valid {
			continue
		}
		description := strings.TrimSpace(asStringValue(item["description"]))
		if description == "" {
			continue
		}
		if len(description) > 240 {
			description = strings.TrimSpace(description[:240])
		}
		skillsByName[name] = agentSkillSummary{
			Name:        name,
			Description: description,
		}
	}

	if len(skillsByName) == 0 {
		return []agentSkillSummary{}
	}
	names := make([]string, 0, len(skillsByName))
	for name := range skillsByName {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]agentSkillSummary, 0, len(names))
	for _, name := range names {
		out = append(out, skillsByName[name])
	}
	return out
}

func normalizeSkillName(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if len(normalized) < 2 || len(normalized) > 64 {
		return "", false
	}
	for _, ch := range normalized {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.':
		default:
			return "", false
		}
	}
	return normalized, true
}

func asStringValue(value any) string {
	str, _ := value.(string)
	return str
}

func defaultSkillCallContract(apiBase string) map[string]any {
	return map[string]any{
		"schema_version": "1",
		"transport": map[string]any{
			"channel":          "agent_message",
			"publish_endpoint": apiBase + "/messages/publish",
			"pull_endpoint":    apiBase + "/messages/pull",
		},
		"request": map[string]any{
			"type":            "skill_request",
			"required_fields": []string{"type", "request_id", "skill_name", "reply_required"},
			"field_notes": map[string]any{
				"request_id":          "caller-generated correlation id",
				"skill_name":          "must match the peer's advertised skill name",
				"reply_required":      "boolean; set true when a result must be returned",
				"reply_to_agent_id":   "optional preferred logical return target",
				"reply_to_agent_uuid": "optional preferred UUID return target",
				"input":               "optional free-form instruction or parameters",
			},
			"json_example": map[string]any{
				"type":                "skill_request",
				"request_id":          "req-20260317-001",
				"skill_name":          "weather_lookup",
				"reply_required":      true,
				"reply_to_agent_uuid": "11111111-1111-1111-1111-111111111111",
				"input":               "Seattle, WA",
			},
			"markdown_example": strings.Join([]string{
				"type: skill_request",
				"request_id: req-20260317-001",
				"skill_name: weather_lookup",
				"reply_required: true",
				"reply_to_agent_uuid: 11111111-1111-1111-1111-111111111111",
				"input: Seattle, WA",
			}, "\n"),
		},
		"result": map[string]any{
			"type":            "skill_result",
			"required_fields": []string{"type", "request_id", "skill_name", "status", "output"},
			"field_notes": map[string]any{
				"request_id": "must match the incoming skill_request.request_id",
				"status":     "ok or error",
				"output":     "result body from executed skill",
				"error":      "optional error detail when status=error",
			},
			"json_example": map[string]any{
				"type":       "skill_result",
				"request_id": "req-20260317-001",
				"skill_name": "weather_lookup",
				"status":     "ok",
				"output":     "Seattle 8C and overcast",
			},
			"markdown_example": strings.Join([]string{
				"type: skill_result",
				"request_id: req-20260317-001",
				"skill_name: weather_lookup",
				"status: ok",
				"output: Seattle 8C and overcast",
			}, "\n"),
		},
	}
}
