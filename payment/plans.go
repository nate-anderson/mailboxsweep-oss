package payment

import (
	_ "embed"
	"mailboxsweep/inboxuser"

	"gopkg.in/yaml.v3"
)

type PlanDetails struct {
	PriceInCents int      `yaml:"price_in_cents"`
	DisplayName  string   `yaml:"display_name"`
	Description  string   `yaml:"description"`
	Features     []string `yaml:"features"`
	Enabled      bool     `yaml:"enabled"`
}

//go:embed plans.yml
var planData string

func readPlans() (map[inboxuser.Plan]PlanDetails, error) {
	m := make(map[inboxuser.Plan]PlanDetails)
	err := yaml.Unmarshal([]byte(planData), &m)
	return m, err
}
