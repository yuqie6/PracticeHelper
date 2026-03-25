package vectorstore

import (
	"fmt"
	"strings"
	"time"
)

const (
	ProviderQdrant = "qdrant"
)

func NewStore(
	provider string,
	baseURL string,
	apiKey string,
	collection string,
	timeout time.Duration,
) (Store, error) {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	trimmedURL := strings.TrimSpace(baseURL)
	if normalizedProvider == "" && trimmedURL == "" {
		return nil, nil
	}
	if normalizedProvider == "" {
		normalizedProvider = ProviderQdrant
	}

	switch normalizedProvider {
	case "disabled", "none", "off":
		return nil, nil
	case ProviderQdrant:
		if trimmedURL == "" {
			return nil, fmt.Errorf(
				"vector store provider %q requires PRACTICEHELPER_SERVER_VECTOR_STORE_URL",
				ProviderQdrant,
			)
		}
		if strings.TrimSpace(collection) == "" {
			return nil, fmt.Errorf(
				"vector store provider %q requires PRACTICEHELPER_SERVER_VECTOR_STORE_COLLECTION",
				ProviderQdrant,
			)
		}
		return NewQdrantStore(trimmedURL, apiKey, collection, timeout), nil
	default:
		return nil, fmt.Errorf(
			"unsupported vector store provider %q; currently only %q is implemented",
			normalizedProvider,
			ProviderQdrant,
		)
	}
}
