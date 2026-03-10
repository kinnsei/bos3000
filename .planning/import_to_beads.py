#!/usr/bin/env python3
"""Import GSD plans into beads-rust with proper dependency DAG."""

import json
import os
import re
import subprocess
import sys
from pathlib import Path
from datetime import datetime, timezone

PLANNING_DIR = Path(".planning")
PHASES_DIR = PLANNING_DIR / "phases"

# Phase directories in order
PHASE_DIRS = sorted(PHASES_DIR.iterdir()) if PHASES_DIR.exists() else []

# Priority mapping
def priority_for_phase(phase_num: int) -> int:
    if phase_num <= 2:
        return 1
    elif phase_num <= 5:
        return 2
    return 3

def run_br(args: list[str], check=True) -> dict | None:
    """Run br command and return parsed JSON output."""
    cmd = ["br"] + args
    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0 and check:
        print(f"ERROR: br {' '.join(args)} failed: {result.stderr}", file=sys.stderr)
        return None
    if "--json" in args:
        try:
            return json.loads(result.stdout)
        except json.JSONDecodeError:
            print(f"ERROR: Invalid JSON from br {' '.join(args)}: {result.stdout}", file=sys.stderr)
            return None
    return {"stdout": result.stdout, "stderr": result.stderr}

def parse_frontmatter(content: str) -> dict:
    """Parse YAML-like frontmatter from PLAN.md."""
    fm = {}
    match = re.match(r'^---\n(.*?)\n---', content, re.DOTALL)
    if not match:
        return fm

    for line in match.group(1).split('\n'):
        line = line.strip()
        if ':' in line and not line.startswith('-') and not line.startswith('#'):
            key, _, val = line.partition(':')
            key = key.strip()
            val = val.strip()
            if key in ('wave', 'plan'):
                try:
                    fm[key] = int(val)
                except ValueError:
                    fm[key] = val
            elif key == 'depends_on':
                # Parse array like ["01-01", "01-02"]
                deps = re.findall(r'"([^"]+)"', val)
                fm[key] = deps
            elif key == 'autonomous':
                fm[key] = val.lower() == 'true'
            elif key == 'phase':
                fm[key] = val
            else:
                fm[key] = val
    return fm

def extract_tasks(content: str) -> list[dict]:
    """Extract task elements from PLAN.md content."""
    tasks = []
    # Find all <task ...> ... </task> blocks
    task_pattern = re.compile(r'<task\s+(.*?)>(.*?)</task>', re.DOTALL)

    for match in task_pattern.finditer(content):
        attrs_str = match.group(1)
        body = match.group(2)

        task = {}

        # Parse attributes
        type_match = re.search(r'type="(\w+)"', attrs_str)
        task['type'] = type_match.group(1) if type_match else 'auto'

        # Extract fields
        for tag in ['name', 'action', 'files', 'verify', 'done']:
            tag_match = re.search(rf'<{tag}>(.*?)</{tag}>', body, re.DOTALL)
            if tag_match:
                task[tag] = tag_match.group(1).strip()
            else:
                # Try nested tags for verify
                if tag == 'verify':
                    auto_match = re.search(r'<automated>(.*?)</automated>', body, re.DOTALL)
                    if auto_match:
                        task['verify'] = auto_match.group(1).strip()

        # Extract task number from name
        num_match = re.search(r'Task\s+(\d+)', task.get('name', ''))
        task['num'] = int(num_match.group(1)) if num_match else 0

        # Clean name - remove "Task N: " prefix
        name = task.get('name', 'Unknown')
        name = re.sub(r'^Task\s+\d+:\s*', '', name)
        task['clean_name'] = name

        tasks.append(task)

    return tasks

def build_description(task: dict, phase_num: int, wave: int, task_num: int) -> str:
    """Build beads issue description from task data."""
    parts = []

    if task.get('action'):
        parts.append(f"## Action\n\n{task['action']}")

    if task.get('files'):
        parts.append(f"## Files\n\n{task['files']}")

    if task.get('verify'):
        parts.append(f"## Verification\n\n{task['verify']}")

    if task.get('done'):
        parts.append(f"## Done When\n\n{task['done']}")

    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    parts.append(f"---\n_Source: GSD phase {phase_num}, wave {wave}, task {task_num}_\n_Import: {now}_")

    return "\n\n".join(parts)


def main():
    failures = []

    # Milestone name from ROADMAP
    milestone_name = "BOS3000 API 回拨双呼系统"

    # Phase info from directory names
    phases = []
    for d in PHASE_DIRS:
        if not d.is_dir():
            continue
        name = d.name
        num_match = re.match(r'^(\d+)-(.+)$', name)
        if num_match:
            phases.append({
                'num': int(num_match.group(1)),
                'slug': num_match.group(2),
                'dir': d,
                'name': name,
            })

    phases.sort(key=lambda p: p['num'])
    print(f"Found {len(phases)} phases")

    # Create top-level epic
    result = run_br(["create", f"GSD: {milestone_name}", "--type", "epic", "--priority", "1", "--json"])
    if not result:
        print("FATAL: Could not create top-level epic", file=sys.stderr)
        sys.exit(1)
    epic_id = result.get('id')
    print(f"Epic: {epic_id} — {milestone_name}")

    # Create phase sub-epics
    phase_ids = {}
    phase_labels = {
        1: "平台基础",
        2: "呼叫引擎（Mock）",
        3: "FreeSWITCH + 录音 + Webhook",
        4: "Admin Dashboard",
        5: "Client Portal",
    }

    for phase in phases:
        pnum = phase['num']
        label = phase_labels.get(pnum, phase['slug'])
        title = f"[P{pnum:02d}] {label}"
        result = run_br(["create", title, "--type", "epic", "--priority", "1", "--parent", epic_id, "--json"])
        if result:
            phase_ids[pnum] = result['id']
            print(f"  Phase {pnum}: {result['id']} — {label}")
        else:
            failures.append(f"Phase epic {pnum}")

    # Wire sequential phase dependencies
    phase_nums = sorted(phase_ids.keys())
    for i in range(1, len(phase_nums)):
        prev = phase_ids[phase_nums[i-1]]
        curr = phase_ids[phase_nums[i]]
        run_br(["dep", "add", curr, prev], check=False)
        print(f"  dep: P{phase_nums[i]:02d} -> P{phase_nums[i-1]:02d}")

    # Parse all PLAN.md files and create tasks
    # Track task IDs for dependency wiring
    # Key: "PP-WW-TT" (phase-wave-tasknum), Value: beads ID
    task_id_map = {}
    # Track tasks per plan for inter-plan deps
    # Key: "PP-NN" (phase-plannum), Value: list of (wave, task_ids)
    plan_tasks = {}
    # Track waves per phase
    # Key: phase_num, Value: {wave_num: [task_ids]}
    phase_waves = {}
    # Track plan ordering per phase
    # Key: phase_num, Value: [(plan_num, wave_num)]
    phase_plan_order = {}
    # Track checkpoint count
    checkpoint_count = 0
    dep_count = 0

    for phase in phases:
        pnum = phase['num']
        phase_dir = phase['dir']
        phase_id = phase_ids.get(pnum)
        if not phase_id:
            continue

        phase_waves[pnum] = {}
        phase_plan_order[pnum] = []

        # Find all PLAN.md files in this phase
        plan_files = sorted(phase_dir.glob("*-PLAN.md"))

        for plan_file in plan_files:
            content = plan_file.read_text()
            fm = parse_frontmatter(content)

            plan_num = fm.get('plan', 0)
            if isinstance(plan_num, str):
                try:
                    plan_num = int(plan_num)
                except ValueError:
                    plan_num = 0

            wave = fm.get('wave', 1)
            if isinstance(wave, str):
                try:
                    wave = int(wave)
                except ValueError:
                    wave = 1

            is_checkpoint = fm.get('autonomous') == False and fm.get('type') != 'execute'
            depends_on = fm.get('depends_on', [])

            phase_plan_order[pnum].append((plan_num, wave, depends_on))

            tasks = extract_tasks(content)
            if not tasks:
                print(f"  WARN: No tasks in {plan_file.name}")
                continue

            plan_key = f"{pnum:02d}-{plan_num:02d}"
            plan_tasks[plan_key] = []

            if wave not in phase_waves[pnum]:
                phase_waves[pnum][wave] = []

            priority = priority_for_phase(pnum)

            for task in tasks:
                tnum = task['num']
                tag = f"[P{pnum:02d}-W{wave}-T{tnum:02d}]"
                clean_name = task['clean_name']

                # Check if checkpoint (plan-level or task-level)
                is_task_checkpoint = task.get('type') == 'checkpoint' or (is_checkpoint and not fm.get('autonomous', True))

                if is_task_checkpoint:
                    title = f"{tag} [CHECKPOINT] {clean_name}"
                    checkpoint_count += 1
                else:
                    title = f"{tag} {clean_name}"

                desc = build_description(task, pnum, wave, tnum)

                # Truncate description if too long (br may have limits)
                if len(desc) > 8000:
                    desc = desc[:7900] + "\n\n...(truncated)"

                result = run_br(["create", title, "--type", "task", "--priority", str(priority),
                                "--parent", phase_id, "--description", desc, "--json"], check=False)

                if result and result.get('id'):
                    tid = result['id']
                    task_key = f"{pnum:02d}-{wave}-{tnum:02d}"
                    task_id_map[task_key] = tid
                    plan_tasks[plan_key].append((wave, tnum, tid))
                    phase_waves[pnum][wave].append(tid)

                    # Block checkpoint tasks
                    if is_task_checkpoint:
                        run_br(["block", tid, "Checkpoint: requires human review before proceeding"], check=False)

                    print(f"    {tag} {clean_name[:40]}... -> {tid}")
                else:
                    failures.append(f"{tag} {clean_name}")
                    print(f"    FAIL: {tag} {clean_name}")

    # Wire dependencies
    print("\n=== Wiring Dependencies ===")

    # 6a: Intra-phase wave dependencies
    for pnum in sorted(phase_waves.keys()):
        waves = phase_waves[pnum]
        wave_nums = sorted(waves.keys())

        for i in range(1, len(wave_nums)):
            prev_wave = wave_nums[i-1]
            curr_wave = wave_nums[i]
            prev_tasks = waves[prev_wave]
            curr_tasks = waves[curr_wave]

            if len(prev_tasks) <= 3:
                # Full Cartesian for small waves
                for ct in curr_tasks:
                    for pt in prev_tasks:
                        r = run_br(["dep", "add", ct, pt], check=False)
                        dep_count += 1
            else:
                # Fan-in: last task of previous wave
                gate = prev_tasks[-1]
                for ct in curr_tasks:
                    r = run_br(["dep", "add", ct, gate], check=False)
                    dep_count += 1

            print(f"  P{pnum:02d}: W{prev_wave} -> W{curr_wave} ({len(prev_tasks)} -> {len(curr_tasks)} tasks)")

    # 6c: Inter-phase boundary dependencies
    for i in range(1, len(phase_nums)):
        prev_pnum = phase_nums[i-1]
        curr_pnum = phase_nums[i]

        prev_waves = phase_waves.get(prev_pnum, {})
        curr_waves = phase_waves.get(curr_pnum, {})

        if prev_waves and curr_waves:
            last_wave = max(prev_waves.keys())
            first_wave = min(curr_waves.keys())

            last_tasks = prev_waves[last_wave]
            first_tasks = curr_waves[first_wave]

            if last_tasks and first_tasks:
                # Last task of prev phase -> first tasks of next phase
                gate = last_tasks[-1]
                for ft in first_tasks:
                    run_br(["dep", "add", ft, gate], check=False)
                    dep_count += 1
                print(f"  P{prev_pnum:02d} -> P{curr_pnum:02d} boundary ({len(first_tasks)} tasks depend on gate)")

    # Sync
    run_br(["sync", "--flush-only"], check=False)

    # Summary
    total_tasks = sum(len(tasks) for tasks in plan_tasks.values())
    total_phases = len(phase_ids)

    # Get ready count
    ready_result = subprocess.run(["br", "ready", "--json"], capture_output=True, text=True)
    try:
        ready_data = json.loads(ready_result.stdout)
        ready_count = len(ready_data)
    except (json.JSONDecodeError, TypeError):
        ready_count = "?"

    print(f"\n=== Import Summary ===")
    print(f"Epic:         {epic_id} — {milestone_name}")
    print(f"Phases:       {total_phases}")
    print(f"Tasks:        {total_tasks} ({checkpoint_count} checkpoints)")
    print(f"Dependencies: {dep_count} links")
    print(f"Ready:        {ready_count} tasks unblocked")
    print(f"Failures:     {len(failures)}")

    if failures:
        print(f"\nFailed tasks:")
        for f in failures:
            print(f"  - {f}")

    # Write summary JSON for the caller
    summary = {
        "epic_id": epic_id,
        "milestone": milestone_name,
        "phases": total_phases,
        "tasks": total_tasks,
        "checkpoints": checkpoint_count,
        "deps": dep_count,
        "ready": ready_count,
        "failures": len(failures),
        "phase_ids": phase_ids,
    }

    with open(PLANNING_DIR / "beads_import_summary.json", "w") as f:
        json.dump(summary, f, indent=2, ensure_ascii=False)

if __name__ == "__main__":
    main()
