package donut

import (
	"regexp"
	"strings"
)

var donutStartTagPattern = regexp.MustCompile(`(?i)<s_(.*?)>`)

// TokensToJSON converts a generated Donut token sequence (e.g. "<s_total>20.00</s_total>") into nested maps and lists,
// the same way as DonutProcessor.token2json of Hugging Face transformers (without categorical special tokens)
func TokensToJSON(tokens string) map[string]any {
	result, _ := tokensToJSON(tokens, false)

	if object, ok := result.(map[string]any); ok {
		return object
	}

	if list, ok := result.([]any); ok && len(list) > 0 {
		if object, ok := list[0].(map[string]any); ok {
			return object
		}
	}

	return map[string]any{}
}

func tokensToJSON(tokens string, isInnerValue bool) (any, string) {
	output := map[string]any{}

	for tokens != "" {
		startMatch := donutStartTagPattern.FindStringSubmatchIndex(tokens)

		if startMatch == nil {
			break
		}

		key := tokens[startMatch[2]:startMatch[3]]
		startToken := tokens[startMatch[0]:startMatch[1]]
		endPattern := regexp.MustCompile(`(?i)</s_` + regexp.QuoteMeta(key) + `>`)
		endMatch := endPattern.FindStringIndex(tokens)

		if endMatch == nil {
			tokens = strings.ReplaceAll(tokens, startToken, "")
			continue
		}

		endToken := tokens[endMatch[0]:endMatch[1]]
		contentPattern := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(startToken) + `(.*?)` + regexp.QuoteMeta(endToken))

		if contentMatch := contentPattern.FindStringSubmatch(tokens); contentMatch != nil {
			content := strings.TrimSpace(contentMatch[1])

			if strings.Contains(content, "<s_") && strings.Contains(content, "</s_") {
				value, _ := tokensToJSON(content, true)

				if list, ok := value.([]any); ok && len(list) > 0 {
					if len(list) == 1 {
						output[key] = list[0]
					} else {
						output[key] = list
					}
				}
			} else {
				leaves := make([]any, 0)

				for _, leaf := range strings.Split(content, "<sep/>") {
					leaves = append(leaves, strings.TrimSpace(leaf))
				}

				if len(leaves) == 1 {
					output[key] = leaves[0]
				} else {
					output[key] = leaves
				}
			}
		}

		index := strings.Index(tokens, endToken)
		tokens = strings.TrimSpace(tokens[index+len(endToken):])

		if strings.HasPrefix(tokens, "<sep/>") {
			rest, _ := tokensToJSON(tokens[len("<sep/>"):], true)
			list := []any{output}

			if restList, ok := rest.([]any); ok {
				list = append(list, restList...)
			}

			return list, ""
		}
	}

	if len(output) > 0 {
		if isInnerValue {
			return []any{output}, ""
		}

		return output, ""
	}

	if isInnerValue {
		return []any{}, ""
	}

	return map[string]any{"text_sequence": tokens}, ""
}
