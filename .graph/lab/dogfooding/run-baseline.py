#!/usr/bin/env python3
import json
import os
import subprocess
import sys
from pathlib import Path

SCRIPT_PATH = Path(__file__).resolve()
REPO_ROOT = SCRIPT_PATH.parents[3]
BASELINE_PATH = REPO_ROOT / 'internal' / 'testdata' / 'lab' / 'dogfooding' / 'baseline-v1.json'
GRAPH_BIN = os.environ.get('GRAPH_BIN', 'graph')
GENERATED_DIR = SCRIPT_PATH.parent / 'generated'


def run_graph(args, output_path):
    command = [GRAPH_BIN, *args, '--output', str(output_path)]
    subprocess.run(command, cwd=REPO_ROOT, check=True)
    with output_path.open('r', encoding='utf-8') as handle:
        return json.load(handle)


def main():
    with BASELINE_PATH.open('r', encoding='utf-8') as handle:
        baseline = json.load(handle)

    GENERATED_DIR.mkdir(parents=True, exist_ok=True)

    init_response = run_graph(['lab', 'init'], GENERATED_DIR / 'init-response.json')

    plan = baseline['run_plan']
    run_args = [
        'lab', 'run',
        '--model', plan['model'],
        '--topology', plan['session_topology'],
        '--seed', str(plan['seed']),
    ]
    if not plan.get('randomized', True):
        run_args.append('--no-randomize')
    for task_id in plan['task_ids']:
        run_args.extend(['--task', task_id])
    for condition_id in plan['condition_ids']:
        run_args.extend(['--condition', condition_id])

    run_response = run_graph(run_args, GENERATED_DIR / 'run-response.json')
    result = run_response['result']
    if 'batch' not in result or result['batch'] is None:
        raise SystemExit('expected batch execution result from graph lab run')

    score_map = {
        (entry['task_id'], entry['condition_id']): entry
        for entry in baseline['illustrative_scores']
    }

    generated_run_ids = []
    evaluation_artifacts = []
    for item in result['batch']['runs']:
        task_condition = (item['task_id'], item['condition_id'])
        scoring = score_map.get(task_condition)
        if scoring is None:
            raise SystemExit(f'missing illustrative score for {task_condition}')
        run_id = item['run_id']
        generated_run_ids.append(run_id)
        evaluation_output = GENERATED_DIR / f'evaluate-{run_id}.json'
        evaluation_artifacts.append(str(evaluation_output.relative_to(REPO_ROOT)))
        run_graph([
            'lab', 'evaluate',
            '--run', run_id,
            '--evaluator', 'automated:repo-dogfooding-baseline-v1',
            f"--success={str(scoring['success']).lower()}",
            '--quality', str(scoring['quality_score']),
            '--resumability', str(scoring['resumability_score']),
            '--human-interventions', str(scoring['human_intervention_count']),
            '--notes', scoring['notes'],
            '--evaluated-at', '2026-04-02T09:30:00Z',
        ], evaluation_output)

    report_args = ['lab', 'report']
    for run_id in generated_run_ids:
        report_args.extend(['--run', run_id])
    report_response = run_graph(report_args, GENERATED_DIR / 'report-response.json')

    summary = {
        'schema_version': 'v1',
        'experiment_id': baseline['experiment_id'],
        'init_output': str((GENERATED_DIR / 'init-response.json').relative_to(REPO_ROOT)),
        'run_output': str((GENERATED_DIR / 'run-response.json').relative_to(REPO_ROOT)),
        'generated_run_ids': generated_run_ids,
        'evaluation_outputs': evaluation_artifacts,
        'report_output': str((GENERATED_DIR / 'report-response.json').relative_to(REPO_ROOT)),
        'report_artifact_path': report_response['result']['artifact_path'],
        'limitations': baseline['limitations'],
    }
    json.dump(summary, sys.stdout, indent=2)
    sys.stdout.write('\n')


if __name__ == '__main__':
    main()
