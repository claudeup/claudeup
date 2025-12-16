// ABOUTME: Category detection and plugin mapping for marketplaces
// ABOUTME: Hardcoded support for known marketplaces (wshobson/agents)
package profile

// Category represents a plugin category in a marketplace
type Category struct {
	Name        string
	Description string
	Plugins     []string
}

// HasCategories returns true if the marketplace has category metadata
func HasCategories(marketplaceRepo string) bool {
	// V1: Hardcoded list of known marketplaces with categories
	knownMarketplaces := map[string]bool{
		"wshobson/agents": true,
	}
	return knownMarketplaces[marketplaceRepo]
}

// GetCategories returns available categories for a marketplace
func GetCategories(marketplaceRepo string) []Category {
	// V1: Hardcoded categories for wshobson/agents
	if marketplaceRepo == "wshobson/agents" {
		return []Category{
			{
				Name:        "Backend Development",
				Description: "API design, architecture, GraphQL, Temporal workflows",
				Plugins: []string{
					"backend-development:backend-architect",
					"backend-development:graphql-architect",
					"backend-development:temporal-python-pro",
					"backend-development:tdd-orchestrator",
				},
			},
			{
				Name:        "Frontend Development",
				Description: "React, mobile, UI/UX design",
				Plugins: []string{
					"frontend-mobile-development:frontend-developer",
					"frontend-mobile-development:mobile-developer",
				},
			},
			{
				Name:        "Full Stack Orchestration",
				Description: "Deployment, performance, security, testing",
				Plugins: []string{
					"full-stack-orchestration:deployment-engineer",
					"full-stack-orchestration:performance-engineer",
					"full-stack-orchestration:security-auditor",
					"full-stack-orchestration:test-automator",
				},
			},
			{
				Name:        "Code Quality",
				Description: "Code review, refactoring, documentation",
				Plugins: []string{
					"code-refactoring:code-reviewer",
					"code-refactoring:legacy-modernizer",
					"code-documentation:docs-architect",
					"code-documentation:tutorial-engineer",
				},
			},
			{
				Name:        "Debugging",
				Description: "Error debugging, systematic debugging",
				Plugins: []string{
					"debugging-toolkit:debugger",
					"error-debugging:error-detective",
				},
			},
		}
	}

	return []Category{}
}
