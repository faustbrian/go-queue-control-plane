.PHONY: benchmarks browser disaster-recovery-postgres docs integration-postgres integration-queue

benchmarks:
	scripts/benchmarks.sh

browser:
	NPM_CONFIG_CACHE="$${GOTMPDIR}/npm-cache" npm --prefix _browser ci --ignore-scripts --no-audit --no-fund
	PLAYWRIGHT_BROWSERS_PATH="$${GOTMPDIR}/playwright" _browser/node_modules/.bin/playwright install chromium
	NPM_CONFIG_CACHE="$${GOTMPDIR}/npm-cache" npm --prefix _browser run check:browser
	PLAYWRIGHT_BROWSERS_PATH="$${GOTMPDIR}/playwright" NPM_CONFIG_CACHE="$${GOTMPDIR}/npm-cache" npm --prefix _browser run test:browser

disaster-recovery-postgres:
	scripts/disaster-recovery-postgres.sh

docs:
	scripts/check-docs.sh

integration-postgres:
	scripts/integration-postgres.sh

integration-queue:
	scripts/integration-queue.sh
