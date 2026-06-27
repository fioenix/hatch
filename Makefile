# Root Makefile — overclaud skill packaging.
# (Hatch's Go build lives in hatch/Makefile.)

SKILL := overclaud.skill

.PHONY: skill clean-skill help

skill: ## Package skill/ into overclaud.skill (reproducible build artifact)
	rm -f $(SKILL)
	cd . && zip -r -X -q $(SKILL) skill -x '*/.DS_Store' '.DS_Store'
	@echo "Built $(SKILL):"
	@unzip -l $(SKILL) | tail -1

clean-skill: ## Remove the built skill artifact
	rm -f $(SKILL)

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  %-12s %s\n", $$1, $$2}'
