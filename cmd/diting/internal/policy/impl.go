package policy

import (
	"context"
	"sync"

	"diting/internal/models"
)

// EngineImpl 内置策略引擎：从 YAML 规则文件加载，按顺序匹配返回 Allow/Deny/Review。
type EngineImpl struct {
	mu    sync.RWMutex
	rules []Rule
	path  string
}

// NewEngineImpl 根据规则文件路径创建引擎；path 为空时无规则，默认拒绝。
func NewEngineImpl(rulesPath string) (*EngineImpl, error) {
	e := &EngineImpl{path: rulesPath}
	rules, err := LoadRules(rulesPath)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.rules = rules
	e.mu.Unlock()
	return e, nil
}

// Reload 重新加载规则文件（可用于 SIGHUP 热加载）。
func (e *EngineImpl) Reload() error {
	rules, err := LoadRules(e.path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.rules = rules
	e.mu.Unlock()
	return nil
}

// Evaluate 按规则顺序匹配，第一条命中即返回对应 Decision；无命中则 Deny。
func (e *EngineImpl) Evaluate(ctx context.Context, req *models.RequestContext) (*models.Decision, error) {
	subject := req.AgentIdentity
	if subject == "" {
		subject = "*"
	}
	action := req.Action
	if action == "" {
		action = req.Method
	}
	resource := req.Resource
	if resource == "" {
		resource = req.TargetURL
	}
	if resource == "" {
		resource = "*"
	}

	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	for i := range rules {
		r := &rules[i]
		if r.Match(subject, action, resource) {
			reason := r.Reason
			if reason == "" {
				reason = string(r.Decision) + " by rule " + r.ID
			}
			ruleID := r.ID
			if ruleID == "" {
				ruleID = "rule_" + string(r.Decision)
			}
			switch r.Decision {
			case RuleAllow:
				return &models.Decision{
					Kind:           models.DecisionAllow,
					PolicyRuleID:   ruleID,
					DecisionReason: reason,
				}, nil
			case RuleDeny:
				return &models.Decision{
					Kind:           models.DecisionDeny,
					PolicyRuleID:   ruleID,
					DecisionReason: reason,
				}, nil
			case RuleReview:
				return &models.Decision{
					Kind:           models.DecisionReview,
					PolicyRuleID:   ruleID,
					DecisionReason: reason,
				}, nil
			default:
				continue
			}
		}
	}
	// 默认拒绝
	return &models.Decision{
		Kind:           models.DecisionDeny,
		PolicyRuleID:   "default",
		DecisionReason: "no matching rule, default deny",
	}, nil
}

// 编译期保证 EngineImpl 实现 Engine。
var _ Engine = (*EngineImpl)(nil)
