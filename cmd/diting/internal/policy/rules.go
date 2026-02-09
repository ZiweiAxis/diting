// Package policy 规则文件格式与加载。
package policy

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RuleDecision 规则决策：allow / deny / review。
type RuleDecision string

const (
	RuleAllow  RuleDecision = "allow"
	RuleDeny   RuleDecision = "deny"
	RuleReview RuleDecision = "review"
)

// Rule 单条策略规则；空字符串表示通配。
type Rule struct {
	ID       string       `yaml:"id"`
	Subject  string       `yaml:"subject,omitempty"`  // 空或 * 表示任意
	Action   string       `yaml:"action,omitempty"`  // 空或 * 表示任意
	Resource string       `yaml:"resource,omitempty"` // 空或 * 表示任意
	Decision RuleDecision `yaml:"decision"`
	Reason   string       `yaml:"reason,omitempty"` // 决策理由，写入审计
}

// RulesFile 规则文件根结构。
type RulesFile struct {
	Rules []Rule `yaml:"rules"`
}

// LoadRules 从 path 加载 YAML 规则文件；若文件不存在或为空则返回空列表。
func LoadRules(path string) ([]Rule, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("policy rules read: %w", err)
	}
	var f RulesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("policy rules unmarshal: %w", err)
	}
	return f.Rules, nil
}

// Match 返回 rule 是否匹配 subject/action/resource；空或 * 表示匹配任意。
func (r *Rule) Match(subject, action, resource string) bool {
	match := func(pat, v string) bool {
		return pat == "" || pat == "*" || pat == v
	}
	return match(r.Subject, subject) && match(r.Action, action) && match(r.Resource, resource)
}
