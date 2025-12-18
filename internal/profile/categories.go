// ABOUTME: Category detection and plugin mapping for marketplaces
// ABOUTME: Hardcoded support for known marketplaces (wshobson/agents)
package profile

import "strings"

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
	// Source: internal/profile/profiles/scripts/hobson/hobson-setup.sh
	if marketplaceRepo == "wshobson/agents" {
		return []Category{
			{
				Name:        "Core Development",
				Description: "workflows, debugging, docs, refactoring",
				Plugins:     parsePlugins("code-documentation debugging-toolkit git-pr-workflows backend-development frontend-mobile-development full-stack-orchestration code-refactoring dependency-management error-debugging team-collaboration documentation-generation c4-architecture multi-platform-apps developer-essentials"),
			},
			{
				Name:        "Quality & Testing",
				Description: "code review, testing, cleanup",
				Plugins:     parsePlugins("unit-testing tdd-workflows code-review-ai comprehensive-review performance-testing-review framework-migration codebase-cleanup"),
			},
			{
				Name:        "AI & Machine Learning",
				Description: "LLM dev, agents, MLOps",
				Plugins:     parsePlugins("llm-application-dev agent-orchestration context-management machine-learning-ops"),
			},
			{
				Name:        "Infrastructure & DevOps",
				Description: "K8s, cloud, CI/CD, monitoring",
				Plugins:     parsePlugins("deployment-strategies deployment-validation kubernetes-operations cloud-infrastructure cicd-automation incident-response error-diagnostics distributed-debugging observability-monitoring"),
			},
			{
				Name:        "Security & Compliance",
				Description: "scanning, compliance, API security",
				Plugins:     parsePlugins("security-scanning security-compliance backend-api-security frontend-mobile-security"),
			},
			{
				Name:        "Data & Databases",
				Description: "ETL, schema design, migrations",
				Plugins:     parsePlugins("data-engineering data-validation-suite database-design database-migrations application-performance database-cloud-optimization"),
			},
			{
				Name:        "Languages",
				Description: "Python, JS/TS, Go, Rust, etc.",
				Plugins:     parsePlugins("python-development javascript-typescript systems-programming jvm-languages web-scripting functional-programming julia-development arm-cortex-microcontrollers shell-scripting"),
			},
			{
				Name:        "Business & Specialty",
				Description: "SEO, analytics, blockchain, gaming",
				Plugins:     parsePlugins("api-scaffolding api-testing-observability seo-content-creation seo-technical-optimization seo-analysis-monitoring business-analytics hr-legal-compliance customer-sales-automation content-marketing blockchain-web3 quantitative-trading payment-processing game-development accessibility-compliance"),
			},
		}
	}

	return []Category{}
}

// parsePlugins converts space-separated plugin string to slice
func parsePlugins(s string) []string {
	return strings.Fields(s)
}
