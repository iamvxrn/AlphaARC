"""Pinned referee audit plus a finite existence witness. Standard library only."""
import argparse
from collections import Counter, defaultdict
from fractions import Fraction
import hashlib
import itertools
import json
from pathlib import Path
import subprocess
import time

RULES = tuple(itertools.permutations(range(4)))


def optimum(data_and_labels):
    groups = defaultdict(Counter)
    for data, label in data_and_labels:
        groups[data][label] += 1
    return (Fraction(sum(max(g.values()) for g in groups.values()), 24),
            len(groups))


def audit_v0(repo):
    source = subprocess.check_output(
        ['git', 'show', 'c039d2a:referee/referee.py'], cwd=repo, text=True)
    module = {'__name__': 'pinned_referee_v0'}
    exec(compile(source, 'c039d2a:referee/referee.py', 'exec'), module)
    designs = tuple(module['enumerate_family']())
    minima, scores, per_mod = Counter(), Counter(), Counter()
    for design in designs:
        best, rows = module['judge'](design)
        assert best is not None
        minima[str(Fraction(best[0]))] += 1
        scores.update(str(Fraction(row[0])) for row in rows)
        per_mod[design['mod']] += 1
    assert len(designs) == sum(2*m + m*m for m in (3,4,5)) == 74
    assert minima == {'1/2': 74}
    assert scores == {'1/2': 1384, '1': 3592}
    return {'referee_git_commit': 'c039d2a',
            'referee_sha256': hashlib.sha256(source.encode()).hexdigest(),
            'designs': len(designs), 'designs_per_modulus': dict(per_mod),
            'per_design_minimum_distribution': dict(minima),
            'all_probe_maximum_distribution': dict(scores),
            'total_design_probe_pairs': sum(scores.values())}


def independent_dial_counterexample():
    # Own transition/distance expressions; no imports from the referee.
    vectors = ((1,0), (3,0), (0,1), (0,3))
    def successor(s, effect):
        u,v = vectors[effect]
        return ((s[0]+u)%4, (s[1]+v)%4, s[2], s[3])
    def distance(s):
        return sum(min((s[i]-s[i+2])%4, (s[i+2]-s[i])%4) for i in (0,1))
    probe = (0,1,0,2)
    identity_votes, hits = None, 0
    for rule in RULES:
        votes = [0]*4
        for d1,t1 in itertools.product(range(4), repeat=2):
            s = (d1,1,t1,0)
            for a in range(4):
                votes[a] += distance(successor(s,rule[a])) < distance(s)
        zeros = [a for a in range(4) if votes[a] == 0]
        assert len(zeros) == 1
        hits += distance(successor(probe,rule[zeros[0]])) == 0
        if rule == (0,1,2,3):
            identity_votes = votes
    assert identity_votes == [8,8,0,16] and hits == 24
    return {'probe': probe, 'key': ['d2'], 'identity_votes': identity_votes,
            'correct_rule_instances': hits, 'rule_instances': 24, 'accuracy': '1'}


def existence_witness():
    start, probe = (4,4), (5,0)
    subsets = tuple(k for n in range(3) for k in itertools.combinations(range(2),n))
    def project(s,k):
        return tuple(s[i] for i in k)
    def transition(s,a,r):
        return (r[a],s[1])
    def goal(s):
        return s[0] in range(4) and (s[1] == 4 or s[0] == s[1])
    def potential(s):
        return int(not goal(s))
    datasets = {}
    for r in RULES:
        rows = tuple((start,a,transition(start,a,r)) for a in range(4))
        assert all(potential(after) < potential(before) for before,a,after in rows)
        assert all(probe != s for before,a,after in rows for s in (before,after))
        assert sum(goal(transition(probe,a,r)) for a in range(4)) == 1
        datasets[r] = rows

    result = []
    for statistic in ('counts','proj_triples'):
        for k in subsets:
            pairs = []
            for r in RULES:
                rows = [row for row in datasets[r] if project(row[0],k)==project(probe,k)]
                if not rows:
                    data = None
                elif statistic == 'counts':
                    data = tuple(sum(a == label for before,a,after in rows) for label in range(4))
                else:
                    data = tuple(sorted((project(before,k),a,project(after,k))
                                        for before,a,after in rows))
                pairs.append((data,r.index(0)))
            value, classes = optimum(pairs)
            assert classes == 1 and value == Fraction(1,4)
            # Independent enumeration of all deterministic policies on this one class.
            brute_force = max(Fraction(sum(r.index(0)==action for r in RULES),24)
                              for action in range(4))
            assert value == brute_force
            result.append({'statistic': statistic, 'key_indices': k,
                           'classes': classes, 'optimal_accuracy': str(value)})

    full, full_classes = optimum((datasets[r],r.index(0)) for r in RULES)
    erased, _ = optimum((None,r.index(0)) for r in RULES)
    explicit_hits = 0
    for r in RULES:
        # Construct the answer from observed successors, without using r at decision time.
        data = datasets[r]
        selected = next(a for before,a,after in data if after[0] == probe[1])
        explicit_hits += goal(transition(probe,selected,r))
    assert full == 1 and full_classes == 24 and erased == Fraction(1,4)
    assert explicit_hits == 24
    return {'acquisition_start': start, 'probe': probe, 'projected_statistics': result,
            'full_history_classes': full_classes, 'full_history_optimum': str(full),
            'explicit_history_policy_hits': explicit_hits,
            'erased_history_optimum': str(erased),
            'scope': 'Finite overwrite family supplied as prior; logical feasibility witness, not a Q1 repair, learning-family discovery, or AGI result'}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--repo', type=Path, default=Path('/home/aiden/extra/git/AlphaARC'))
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.output.exists():
        parser.error('Choose a new output path; prior results are preserved')
    start = time.perf_counter()
    results = {'status': 'RUNNING', 'seed_count': 0, 'sampling_noise': 0,
               'plan_sha256': '73300031a5c5871e0573dd3481b6a5f8a8f4207c00f15c0ad9e1cb5292e72067',
               'script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
    try:
        results['referee_v0_audit'] = audit_v0(args.repo)
        results['independent_dial_counterexample'] = independent_dial_counterexample()
        results['existence_witness'] = existence_witness()
        results['status'] = 'PASS'
    except Exception as exc:
        results['status'] = 'FAILURE_FROZEN'
        results['error'] = repr(exc)
        raise
    finally:
        results['elapsed_seconds'] = time.perf_counter() - start
        args.output.write_text(json.dumps(results,indent=2)+'\n')
        print(json.dumps(results,indent=2))


if __name__ == '__main__':
    main()
