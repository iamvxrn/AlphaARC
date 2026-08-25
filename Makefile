# AlphaARC -- the score loop.
#
#   make bundle   python/alphaarc/ -> the kit's single-file agent/my_agent.py
#   make quick    INNER LOOP: the 4 games that ever score (~5 min, not ~40)
#   make bench    bundle + play the TRAIN split offline, print the official score
#   make holdout  the same on the FROZEN holdout -- to measure, never to tune
#   make census   classify each TRAIN game's transition kinds (what to model next)
#   make test     Go + Python suites
#
# The Kaggle number is `mean over games of  sum(level_score_i * i) / sum(i over
# all levels)`, level_score = min(115, (baseline/actions)^2 * 100). Depth and
# efficiency both count, quadratically for efficiency -- so `make bench` reports
# per-level actions/baseline, not just "took L1".

KIT      ?= $(HOME)/ARC-AGI-3-Kaggle-Starter/ARC-AGI-3-Kaggle-Starter
PY       := $(KIT)/.venv/bin/python
GO       ?= /run/host/usr/lib/go/bin/go
STEPS    ?= 250
REPEATS  ?= 2
SEED     ?= 1
OUT      ?=
VS       ?=

BENCH_ARGS = --kit $(KIT) --max-steps $(STEPS) --repeats $(REPEATS) --seed $(SEED) \
             $(if $(OUT),--out $(OUT)) $(if $(VS),--vs $(VS))

.PHONY: help quick bundle bench bench-all holdout census test test-go test-py check-bundle

help:
	@sed -n '2,12p' $(MAKEFILE_LIST)

bundle: ## flatten the package into the kit's agent/my_agent.py
	python3 python/bench/bundle.py --kit $(KIT)

check-bundle: ## fail if the built agent is stale w.r.t. the package
	python3 python/bench/bundle.py --kit $(KIT) --check

quick: bundle ## INNER LOOP: only the games that ever score (~5 min, not ~40)
	$(PY) python/bench/bench.py --split scoring $(BENCH_ARGS)

bench: bundle ## train split -- confirmation, after `quick` says something moved
	$(PY) python/bench/bench.py --split train $(BENCH_ARGS)

bench-all: bundle ## every public game (train + holdout); holdout stays unread
	ARC_HOLDOUT_OK=1 $(PY) python/bench/bench.py --split all $(BENCH_ARGS)

holdout: bundle ## the frozen generalization set -- measurement only
	@echo ">>> HOLDOUT: this is our stand-in for the hidden Kaggle set."
	@echo ">>> Read the aggregate. Do NOT open these games to fix a failure."
	ARC_HOLDOUT_OK=1 $(PY) python/bench/bench.py --split holdout $(BENCH_ARGS)

census: ## Phase 1: probe the TRAIN games and classify their transition kinds
	$(PY) python/bench/census.py --split train --clicks 40 --repeats 3 \
	    --out python/bench/runs/census_train.json

test: test-go test-py

test-go:
	$(GO) build ./... && $(GO) vet ./... && $(GO) test ./...

test-py:
	@fail=0; for t in python/tests/test_*.py; do \
	    printf "%-34s " $$t; python3 $$t | tail -1 || fail=1; \
	done; exit $$fail
