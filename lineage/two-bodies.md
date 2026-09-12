CAPABILITY: two-body steering

Observed failure:        m0r0 completes no level on any seed of any measurement,
                         frozen across every run in backstops/. Zero, not small.

Competing hypotheses:    H1 the avatar detector blends two mirrored bodies into a
                            meaningless centroid (recorded in the parent README)
                         H2 no goal region exists on this board -- every detector,
                            enclosure included, returns zero candidates
                         H3 the destination rule is wrong for this game
                         H4 the offset map cannot represent the mechanic

Discriminating evidence: The owner read the board: two markers, one per mirrored
                         half, and the task is to bring them together. Traces
                         agree -- two bodies on 190 of 190 steps, midpoint
                         constant at ~30, distance 20 -> 10 -> 35.
                         Then the decisive one: the router was called 200 times
                         and returned a route SIX times. It had learned two
                         offsets, both vertical, so it could never reach a body
                         offset horizontally. H4 is upstream of H2 and H3.
                         Cause: the horizontal key sends one marker (0,-5) and the
                         other (0,+5), so their union is not a translation and
                         rigid_body records nothing at all.

Predicted intervention:  learning the offset from ONE body, and aiming it at the
                         other, should let m0r0 close the distance. Single-
                         component games must NOT change, since the branch is
                         gated on exactly two components.

Result:                  16 paired seeds, repeats=1, full train split.
                         m0r0  0.0000 -> 0.3868  (+0.3868 +/- 0.0224, 17 se)
                         seeds completing a level: 0 of 16 -> 16 of 16
                         aggregate 0.4800 -> 0.5026 (+0.0226 +/- 0.0013)
                         offsets learned 2 -> 4; routes returned 6/200 -> 114/201
                         Control: g50t, lf52, lp85, ls20, r11l, re86, tn36, vc33
                         all exactly +0.0000 on every seed. ar25 moved -0.0034
                         +/- 0.0033, one standard error, inside the noise -- it
                         presumably shows two components sometimes. Recorded as
                         the single control imperfection rather than dismissed.

Ablation:                ARC_TWO_BODIES=0 restores the failure exactly: m0r0
                         returns to 0.0000 on every seed.

Status:                  REQUIRED
