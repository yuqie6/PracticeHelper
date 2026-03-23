package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"practicehelper/server/internal/domain"
)

const maxPromptOverlayInstructionRunes = 280

var (
	allowedPromptOverlayTones = map[string]string{
		"":           "",
		"supportive": "supportive",
		"direct":     "direct",
		"strict":     "strict",
	}
	allowedPromptOverlayDetailLevels = map[string]string{
		"":         "",
		"concise":  "concise",
		"balanced": "balanced",
		"detailed": "detailed",
	}
	allowedPromptOverlayFollowupIntensity = map[string]string{
		"":         "",
		"light":    "light",
		"standard": "standard",
		"pressure": "pressure",
	}
	allowedPromptOverlayLanguages = map[string]string{
		"":      "",
		"zh-cn": "zh-CN",
		"en-us": "en-US",
	}
	allowedPromptOverlayFocusTags = map[string]struct{}{
		"expression":   {},
		"structure":    {},
		"depth":        {},
		"practicality": {},
		"confidence":   {},
	}
)

func normalizePromptOverlay(input *domain.PromptOverlay) (*domain.PromptOverlay, error) {
	if input == nil {
		return nil, nil
	}

	overlay := &domain.PromptOverlay{
		Tone:              normalizePromptOverlayChoice(input.Tone, allowedPromptOverlayTones),
		DetailLevel:       normalizePromptOverlayChoice(input.DetailLevel, allowedPromptOverlayDetailLevels),
		FollowupIntensity: normalizePromptOverlayChoice(input.FollowupIntensity, allowedPromptOverlayFollowupIntensity),
		AnswerLanguage:    normalizePromptOverlayChoice(input.AnswerLanguage, allowedPromptOverlayLanguages),
		CustomInstruction: strings.TrimSpace(input.CustomInstruction),
	}

	if overlay.Tone == "__invalid__" ||
		overlay.DetailLevel == "__invalid__" ||
		overlay.FollowupIntensity == "__invalid__" ||
		overlay.AnswerLanguage == "__invalid__" {
		return nil, ErrInvalidPromptOverlay
	}

	if len([]rune(overlay.CustomInstruction)) > maxPromptOverlayInstructionRunes {
		return nil, ErrInvalidPromptOverlay
	}

	if len(input.FocusTags) > 0 {
		seen := make(map[string]struct{}, len(input.FocusTags))
		for _, raw := range input.FocusTags {
			tag := strings.TrimSpace(strings.ToLower(raw))
			if tag == "" {
				continue
			}
			if _, ok := allowedPromptOverlayFocusTags[tag]; !ok {
				return nil, ErrInvalidPromptOverlay
			}
			seen[tag] = struct{}{}
		}
		if len(seen) > 2 {
			return nil, ErrInvalidPromptOverlay
		}
		overlay.FocusTags = make([]string, 0, len(seen))
		for tag := range seen {
			overlay.FocusTags = append(overlay.FocusTags, tag)
		}
		slices.Sort(overlay.FocusTags)
	}

	if isEmptyPromptOverlay(overlay) {
		return nil, nil
	}
	return overlay, nil
}

func normalizePromptOverlayChoice(value string, allowed map[string]string) string {
	key := strings.TrimSpace(strings.ToLower(value))
	if normalized, ok := allowed[key]; ok {
		return normalized
	}
	return "__invalid__"
}

func isEmptyPromptOverlay(overlay *domain.PromptOverlay) bool {
	return overlay == nil ||
		(overlay.Tone == "" &&
			overlay.DetailLevel == "" &&
			overlay.FollowupIntensity == "" &&
			overlay.AnswerLanguage == "" &&
			len(overlay.FocusTags) == 0 &&
			overlay.CustomInstruction == "")
}

func promptOverlayHash(overlay *domain.PromptOverlay) string {
	if isEmptyPromptOverlay(overlay) {
		return ""
	}
	payload, err := json.Marshal(overlay)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
