package repo

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func toNullableTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func normalizeDateString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed := parseTime(raw)
	if parsed.IsZero() {
		return raw
	}

	return parsed.UTC().Format(time.RFC3339Nano)
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func newID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buffer))
}

func buildFTSQuery(raw string) string {
	// 这里只做非常保守的 ASCII token 提取，目的是减少 SQLite FTS 的误匹配和奇怪转义问题。
	// 代价是中文、符号和超短 token 的召回会偏弱，所以它只适合作为导入仓库后的轻量检索兜底。
	parts := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && r <= 127
	})
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		filtered = append(filtered, fmt.Sprintf("%q", part))
	}
	return strings.Join(filtered, " OR ")
}

func dedupeWeaknessHits(hits []domain.WeaknessHit) []domain.WeaknessHit {
	type key struct {
		kind  string
		label string
	}

	result := make([]domain.WeaknessHit, 0, len(hits))
	seen := map[key]domain.WeaknessHit{}
	for _, hit := range hits {
		k := key{kind: hit.Kind, label: hit.Label}
		existing, ok := seen[k]
		if !ok || hit.Severity > existing.Severity {
			seen[k] = hit
		}
	}

	for _, hit := range seen {
		result = append(result, hit)
	}

	return result
}

func normalizeMemoryScope(scopeType string) string {
	switch strings.TrimSpace(scopeType) {
	case domain.MemoryScopeProject:
		return domain.MemoryScopeProject
	case domain.MemoryScopeSession:
		return domain.MemoryScopeSession
	case domain.MemoryScopeJobTarget:
		return domain.MemoryScopeJobTarget
	default:
		return domain.MemoryScopeGlobal
	}
}

func normalizeTopicLabel(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}
