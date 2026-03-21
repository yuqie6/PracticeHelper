package service

import (
	"sort"
	"strings"

	"practicehelper/server/internal/domain"
)

var basicsTopics = []string{
	domain.BasicsTopicGo,
	domain.BasicsTopicRedis,
	domain.BasicsTopicKafka,
	domain.BasicsTopicMySQL,
	domain.BasicsTopicSystemDesign,
	domain.BasicsTopicDistributed,
	domain.BasicsTopicNetwork,
	domain.BasicsTopicMicroservice,
	domain.BasicsTopicOS,
	domain.BasicsTopicDockerK8s,
}

var basicsTopicKeywordMap = map[string][]string{
	domain.BasicsTopicGo: {
		"go", "goroutine", "channel", "context", "gmp", "pprof", "gc", "interface",
	},
	domain.BasicsTopicRedis: {
		"redis", "cache", "缓存", "击穿", "雪崩", "穿透", "bigkey", "大key", "lua",
	},
	domain.BasicsTopicKafka: {
		"kafka", "mq", "消息", "consumer", "producer", "offset", "partition", "rebalance",
	},
	domain.BasicsTopicMySQL: {
		"mysql", "sql", "索引", "事务", "mvcc", "binlog", "redo", "死锁", "分库", "分表",
	},
	domain.BasicsTopicSystemDesign: {
		"system design", "架构", "高并发", "限流", "降级", "容量", "扩容", "设计系统",
	},
	domain.BasicsTopicDistributed: {
		"distributed", "分布式", "raft", "paxos", "cap", "幂等", "事务消息",
	},
	domain.BasicsTopicNetwork: {
		"network", "tcp", "http", "https", "udp", "dns", "网络", "握手", "拥塞", "rpc",
	},
	domain.BasicsTopicMicroservice: {
		"microservice", "微服务", "服务治理", "服务发现", "熔断", "网关", "链路追踪",
	},
	domain.BasicsTopicOS: {
		"os", "操作系统", "进程", "线程", "调度", "内核", "内存", "页表", "页缓存", "上下文切换",
	},
	domain.BasicsTopicDockerK8s: {
		"docker", "k8s", "kubernetes", "container", "容器", "pod", "deployment", "cgroup",
	},
}

var defaultMixedTopics = []string{
	domain.BasicsTopicRedis,
	domain.BasicsTopicMySQL,
	domain.BasicsTopicDistributed,
}

type scoredTopic struct {
	name  string
	score float64
}

func selectBasicsTopicsForSession(weaknesses []domain.WeaknessTag, requestedTopic string) []string {
	topic := normalizeBasicsTopic(requestedTopic)
	if topic != domain.BasicsTopicMixed {
		if topic == "" {
			return []string{domain.BasicsTopicGo}
		}
		return []string{topic}
	}

	scores := scoreMixedTopicsFromWeaknesses(weaknesses)
	if len(scores) == 0 {
		return append([]string{}, defaultMixedTopics...)
	}

	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].name < scores[j].name
	})

	selected := make([]string, 0, 3)
	for _, item := range scores {
		selected = append(selected, item.name)
		if len(selected) == 3 {
			break
		}
	}
	if len(selected) == 0 {
		return append([]string{}, defaultMixedTopics...)
	}
	return selected
}

func normalizeBasicsTopic(topic string) string {
	normalized := strings.TrimSpace(strings.ToLower(topic))
	switch normalized {
	case "", domain.BasicsTopicGo, domain.BasicsTopicRedis, domain.BasicsTopicKafka,
		domain.BasicsTopicMySQL, domain.BasicsTopicSystemDesign, domain.BasicsTopicDistributed,
		domain.BasicsTopicNetwork, domain.BasicsTopicMicroservice, domain.BasicsTopicOS,
		domain.BasicsTopicDockerK8s, domain.BasicsTopicMixed:
		return normalized
	default:
		return ""
	}
}

func scoreMixedTopicsFromWeaknesses(weaknesses []domain.WeaknessTag) []scoredTopic {
	if len(weaknesses) == 0 {
		return nil
	}

	scores := map[string]float64{}
	for _, item := range weaknesses {
		base := item.Severity
		if item.Kind == "topic" {
			base += 0.25
		}

		for _, topic := range matchBasicsTopics(item.Label) {
			scores[topic] += base
		}
	}

	if len(scores) == 0 {
		return nil
	}

	items := make([]scoredTopic, 0, len(scores))
	for topic, score := range scores {
		items = append(items, scoredTopic{name: topic, score: score})
	}
	return items
}

func matchBasicsTopics(label string) []string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	if normalized == "" {
		return nil
	}

	if _, ok := basicsTopicKeywordMap[normalized]; ok {
		return []string{normalized}
	}

	seen := map[string]struct{}{}
	matched := make([]string, 0, 2)
	for topic, keywords := range basicsTopicKeywordMap {
		for _, keyword := range keywords {
			if strings.Contains(normalized, keyword) {
				if _, ok := seen[topic]; ok {
					break
				}
				seen[topic] = struct{}{}
				matched = append(matched, topic)
				break
			}
		}
	}
	return matched
}
