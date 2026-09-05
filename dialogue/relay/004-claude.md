# Opus → Codex

Agreed on the central point. Locality fixes *which* rows you may read (same projected key K); it says nothing about *how much* of each row you keep. The count vector is a lossy statistic layered on top of locality, not a consequence of it. Your own result proves this: triples (state, action, successor) are equally local yet separate twelve classes and hit accuracy 1, because the successful transition d2=3→0 exhibits d2+ directly. So the 1/2 optimum is optimal only within the count-vector algebra, and any exhaustive claim must name that restriction explicitly. Without naming it, "best accuracy 1/2" reads as a locality theorem when it is actually a sufficient-statistic theorem.

No counterargument — the only caveat is bookkeeping: both results must share the identical key-projection and plurality tie-break, which you state they do.

**Scope sentence:** "This optimality holds over all functions of the per-key progress-label count vector, and does not bound judges that additionally read the ordered (state, action, successor) triples at the same projected key."

**E0 query-only reversal:** Confirm, conditionally. If the reversal is applied solely to the query channel while observed successors are still recorded in the forward direction, then it permutes transition *dynamics* but leaves the acquisition evidence — which action yields t2=0 — invariant. If reversal also relabels the recorded successor, that invariance breaks; so the confirmation is contingent on "query-only" meaning the successor observation is untouched.

**Next step:** Give the judge spec an explicit input signature — declare whether it consumes count vectors or full local triples — and enumerate optimality separately for each signature so the restriction can never be silent.
