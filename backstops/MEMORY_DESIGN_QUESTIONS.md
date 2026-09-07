# Memory design: the measured constraints, and the five open questions

Not a design. The owner asked for questions sharp enough to answer, and these are
the bounds any answer has to fit. Every number is measured, none inferred.

## The constraints

**C1 — one value per key cannot hold conditionality.** `moves[k2] = (6, 0)` is
right most of the time and there is nowhere in the type to write "except while
the door is shut". See `MEMORY_HAS_NO_ROOM_FOR_CONDITIONALITY.md`.

**C2 — a finer key is not the repair, and this is quantitative.** Three
refutations: `Rejected #6`, conditioning on position (ls20 1% contradictory,
m0r0 80%), and conditioning on body configuration (worse: m0r0 80 → 87%, and
g50t went from **43 repeated pairs to 13**). A finer key partitions the same data
into more cells and loses support without buying truth.

**C3 — the world is mostly lawful.** Contradictions run at **0.58x** what chance
would give at the observed no-op rate. So most controls need no refinement at all;
only a minority do.

**C4 — the sample budget per situation is tiny, and it is inverted.**
Observations per `(control, position)` on g50t:

    seeds that make progress   median 4    21% of situations seen ONCE
                                           half seen 3 times or fewer
    seeds that are stuck       median 9

The agent sees a situation repeatedly only when it is going nowhere. Evidence is
most abundant exactly where it is least useful.

**C5 — no author knowledge.** Contract L4: the decoded mechanics of these games
may not enter through hand-picked structure or features.

## The questions

**Q1. How is a distinction recorded without splitting the data?**
Given C2 and C3: most controls are lawful, so refinement should be *local* --
flat by default, refined only where a contradiction has actually been seen. What
structure supports variable resolution per control rather than one global key?

**Q2. What is the conditioning variable when the thing that changed is invisible?**
The door is not a detectable board feature -- enclosure detection returns zero
regions on sp80 and m0r0. So the variable is either **latent** (an index the agent
invents and assigns by which predictions hold) or a function of **history** rather
than of the current board. Which, and how is it identified from C4's four samples?

**Q3. What is the maximum evidence a distinction may cost?**
State the number before building. Above roughly four observations, the mechanism
can only fire on stuck seeds, which is C4's inversion. Any design whose acceptance
test needs ten samples per cell is dead on arrival.

**Q4. What separates "two modes" from "unreliable"?**
A control that works 70% of the time and a control with two modes are identical in
aggregate. Timing does not separate them: cleanly time-separated contradictions
run at 0.82 of chance on winning seeds and 0.74 on losing ones -- both **below**
chance, see `TEMPORAL_SEPARATION_REFUTED.md`. What observable does?

**Q5. Does the record outlive the episode, and what exactly carries?**
The owner's framing: doors exist across games, so a door-less game does not refute
doors. Under the contract that means the *class* may live in K₀ while the instance
must be K_t. What is the carried object, and what experiment distinguishes
"carried the class" from "carried the instance"?

## The trap

Q1 and Q3 pull against each other. Local refinement spends its samples where the
contradictions are, and by C4 that is where samples are scarcest. A design that
does not say how it survives this is not yet a design.

Status: OPEN SPECIFICATION. NO DESIGN PROPOSED.
