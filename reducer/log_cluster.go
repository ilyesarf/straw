package reducer

import "github.com/ilyesarf/straw/types"

func MaskToken(token string) string {
	if len(token) == 0 {
		return token
	}

	if len(token) >= 2 {
		first := token[0]
		last := token[len(token)-1]
		if (first == '[' && last == ']') || (first == '(' && last == ')') {
			inner := MaskToken(token[1 : len(token)-1])
			if inner == "*" {
				return string(first) + "*" + string(last)
			}
			return string(first) + inner + string(last)
		}
	}

	if looksLikeIP(token) {
		return "*"
	}
	if looksLikeIPCIDR(token) {
		return "*"
	}
	if looksLikeHex(token) {
		return "*"
	}
	if looksLikeNumber(token) {
		return "*"
	}
	if looksLikeNumberWithSuffix(token) {
		return "*"
	}
	if looksLikeDate(token) {
		return "*"
	}

	if i := lastColon(token); i > 0 {
		left := token[:i]
		right := token[i+1:]
		maskedLeft := MaskToken(left)
		maskedRight := MaskToken(right)
		if maskedLeft == "*" || maskedRight == "*" {
			return maskedLeft + ":" + maskedRight
		}
	}

	return token
}

func MaskMessage(msg string) string {
	if len(msg) == 0 {
		return msg
	}

	if msg[0] == '{' {
		return "{...}"
	}

	tokens := splitTokens(msg)
	changed := false
	for i, t := range tokens {
		masked := MaskToken(t)
		if masked != t {
			tokens[i] = masked
			changed = true
		}
	}

	if !changed {
		return msg
	}

	return joinTokens(tokens)
}

func ClusterLogs(logs []types.LogLine) []types.LogCluster {
	counts := make(map[string]int)
	samples := make(map[string]string)

	for _, log := range logs {
		pattern := MaskMessage(log.Message)
		counts[pattern]++
		if _, exists := samples[pattern]; !exists {
			samples[pattern] = log.Message
		}
	}

	clusters := make([]types.LogCluster, 0, len(counts))
	for pattern, count := range counts {
		clusters = append(clusters, types.LogCluster{
			Pattern: pattern,
			Count:   count,
			Sample:  samples[pattern],
		})
	}

	sortClusters(clusters)
	return clusters
}

func sortClusters(clusters []types.LogCluster) {
	for i := 1; i < len(clusters); i++ {
		key := clusters[i]
		j := i - 1
		for j >= 0 && clusters[j].Count < key.Count {
			clusters[j+1] = clusters[j]
			j--
		}
		clusters[j+1] = key
	}
}

func looksLikeIP(s string) bool {
	dots := 0
	digitRun := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			digitRun++
			if digitRun > 3 {
				return false
			}
		} else if c == '.' {
			if digitRun == 0 {
				return false
			}
			dots++
			digitRun = 0
		} else {
			return false
		}
	}
	return dots == 3 && digitRun > 0
}

func looksLikeHex(s string) bool {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			hexCount++
		} else if c == '-' {
			continue
		} else {
			return false
		}
	}
	// 8+ hex chars catches container IDs, UUIDs, commit hashes
	return hexCount >= 8
}

func looksLikeNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	dots := 0
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	for i := start; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' {
			dots++
			if dots > 1 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// matches 0.000217611s, 500ms, 1024kb — common in latency/size fields
func looksLikeNumberWithSuffix(s string) bool {
	if len(s) < 2 {
		return false
	}
	suffixStart := len(s)
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			suffixStart = i
		} else {
			break
		}
	}
	if suffixStart == 0 || suffixStart == len(s) {
		return false
	}
	if len(s)-suffixStart > 4 {
		return false
	}
	return looksLikeNumber(s[:suffixStart])
}

func lastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func looksLikeDate(s string) bool {
	if len(s) != 10 {
		return false
	}
	sep := s[4]
	if sep != '-' && sep != '/' {
		return false
	}
	if s[7] != sep {
		return false
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9} {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func looksLikeIPCIDR(s string) bool {
	slash := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			slash = i
			break
		}
	}
	if slash < 1 || slash >= len(s)-1 {
		return false
	}
	if !looksLikeIP(s[:slash]) {
		return false
	}
	prefix := s[slash+1:]
	if len(prefix) == 0 || len(prefix) > 3 {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if prefix[i] < '0' || prefix[i] > '9' {
			return false
		}
	}
	return true
}

func splitTokens(s string) []string {
	var tokens []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			if start >= 0 {
				tokens = append(tokens, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, s[start:])
	}
	return tokens
}

func joinTokens(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	n := len(tokens) - 1
	for _, t := range tokens {
		n += len(t)
	}
	buf := make([]byte, 0, n)
	for i, t := range tokens {
		if i > 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, t...)
	}
	return string(buf)
}
