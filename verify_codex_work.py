#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
验收 Codex 是否按设计改造 teaching-agents（静态层 + 编译层）。
用法：
    python verify_codex_work.py                      # 默认仓库路径
    python verify_codex_work.py "D:/path/to/repo"    # 指定仓库

输出每项 PASS / FAIL / RED FLAG，最后给总结。
注意：本脚本只判定"代码层是否改到位"；"能不能真的写/查 AI 赛道报告"
必须再做文档里的 UI 手工测试，脚本无法替代。
"""
import os
import re
import subprocess
import sys

DEFAULT_REPO = r"D:/WorkBuddy/2026-11-16-18-31-51/_tmp/teaching-agents-repo"
repo = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_REPO

PASS, FAIL, FLAG = "PASS", "FAIL", "RED FLAG"
results = []


def read(rel):
    p = os.path.join(repo, rel)
    if not os.path.exists(p):
        return None
    with open(p, "r", encoding="utf-8", errors="replace") as f:
        return f.read()


def add(name, status, detail=""):
    results.append((name, status, detail))
    icon = {PASS: "[PASS]", FAIL: "[FAIL]", FLAG: "[RED FLAG]"}[status]
    print(f"{icon} {name}" + (f" -- {detail}" if detail else ""))


print(f"仓库：{repo}\n" + "=" * 60)

# ---------- 0. 分支与提交位置 ----------
branch = ""
try:
    branch = subprocess.check_output(
        ["git", "-C", repo, "branch", "--show-current"], text=True
    ).strip()
except Exception:
    pass
add("0a 改动在独立分支而非 main",
    PASS if branch and branch != "main" else FLAG,
    f"当前分支：{branch or '未知'}（设计要求是 feat/ai-track-builder 之类，不是 main）")

# ---------- 1. embed 是否真加了三个 JSON ----------
builtin = read("builtin.go") or ""
embed_line = ""
for line in builtin.splitlines():
    if "go:embed" in line:
        embed_line = line.strip()
new_files = ["knowledge/rules_7_ai_2026.json",
             "knowledge/ai_playbook.json",
             "knowledge/ai_award_cases_6th.json"]
missing = [f for f in new_files if f not in embed_line]
add("1a builtin.go embed 含三个新 JSON",
    PASS if not missing else FAIL,
    "embed 行：" + (embed_line or "未找到") + (f" | 缺：{missing}" if missing else ""))

# ---------- 1b 关键：embed 之外还有没有真正读取/检索 ----------
go_sources = []
for root, dirs, files in os.walk(repo):
    dirs[:] = [d for d in dirs if d not in (".git", "node_modules", "frontend")]
    for fn in files:
        if fn.endswith(".go"):
            go_sources.append(os.path.join(root, fn))
all_go = ""
for p in go_sources:
    with open(p, "r", encoding="utf-8", errors="replace") as f:
        all_go += f.read() + "\n"

for fname in new_files:
    short = os.path.basename(fname)
    count = all_go.count(short)
    # embed 行里出现 1 次；若总数 <=1，说明只是塞进 embed 没被读取
    add(f"1b {short} 被实际读取/检索",
        PASS if count > 1 else FLAG,
        f"在 .go 文件中出现 {count} 次；<=1 表示只加进 embed 但没有任何加载/检索代码（典型偷懒）")

# ---------- 2. 新顾问 ----------
spec = read("specialists.go") or ""
for sid in ("architect", "video-planner"):
    add(f"2a specialists.go 含顾问 {sid}",
        PASS if re.search(r'["`]' + re.escape(sid) + r'["`]', spec) else FAIL,
        "检查 teachingSpecialists 数组是否新增该 ID")

# 前端下拉是否注册（两种实现都可能：静态列表或动态拉取 /api/specialists）
frontend_hits = []
fe_root = os.path.join(repo, "frontend", "src")
if os.path.isdir(fe_root):
    for root, dirs, files in os.walk(fe_root):
        for fn in files:
            if fn.endswith((".tsx", ".ts", ".jsx", ".js")):
                p = os.path.join(root, fn)
                with open(p, "r", encoding="utf-8", errors="replace") as f:
                    txt = f.read()
                if "architect" in txt or "video-planner" in txt or "specialists" in txt.lower():
                    frontend_hits.append(fn)
add("2b 前端能看到新顾问",
    PASS if frontend_hits else FLAG,
    ("命中：" + ", ".join(sorted(set(frontend_hits)))) if frontend_hits
    else "未在前端源码发现 architect/video-planner 或 specialists 注册；若顾问走 /api/specialists 动态渲染则可人工在 UI 确认")

# ---------- 3. generate 模式 ----------
main_go = read("main.go") or ""
gen_hits = [ln.strip() for ln in main_go.splitlines()
            if re.search(r"generate|Generate|生成模式", ln)]
add("3a main.go 有 generate 模式痕迹",
    PASS if gen_hits else FAIL,
    (f"命中 {len(gen_hits)} 行，例：{gen_hits[0][:90]}" if gen_hits else "未发现 generate/生成模式 关键字"))
# 是否有独立的生成型 system prompt（审阅型三个分支的特征串）
review_markers = ["文字审阅顾问", "图文审阅顾问", "材料结构审阅顾问"]
gen_prompt = [ln.strip()[:90] for ln in main_go.splitlines()
              if "sys =" in ln and not any(m in ln for m in review_markers)]
add("3b 存在新的生成型 system prompt 分支",
    PASS if gen_prompt else FLAG,
    (f"新增 sys 赋值 {len(gen_prompt)} 处" if gen_prompt else
     "除三个审阅分支外没有新的 sys 赋值；可能只是加了字段名但没有生成逻辑"))

# ---------- 4. 测试 ----------
test_files = []
for root, dirs, files in os.walk(repo):
    dirs[:] = [d for d in dirs if d not in (".git", "node_modules")]
    for fn in files:
        if fn.endswith("_test.go"):
            test_files.append(os.path.relpath(os.path.join(root, fn), repo))
test_touched = [t for t in test_files
                if any(k in os.path.basename(t).lower()
                       for k in ("ai_track", "aitrack", "architect", "video", "generate"))]
# 也检查现有测试文件中是否新增了新能力的断言
test_content_hits = []
for p in test_files:
    txt = read(p) or ""
    if any(k in txt for k in ("architect", "video-planner", "ai_award_cases", "generate")):
        test_content_hits.append(p)
add("4a 新增针对性测试文件/断言",
    PASS if (test_touched or test_content_hits) else FLAG,
    (("新文件：" + ", ".join(test_touched)) if test_touched else "")
    + ((" | 含新断言：" + ", ".join(test_content_hits)) if test_content_hits else "")
    or "未发现 AI/architect/video/generate 相关测试文件或断言")

# go test
try:
    proc = subprocess.run(["go", "test", "./..."], cwd=repo,
                          capture_output=True, text=True, timeout=300)
    out = (proc.stdout + proc.stderr).strip()
    tail = "\n".join(out.splitlines()[-8:])
    add("4b go test ./...", PASS if proc.returncode == 0 else FAIL, tail)
except FileNotFoundError:
    add("4b go test ./...", FLAG, "本机未找到 go，跳过；请手动运行 go test ./...")
except subprocess.TimeoutExpired:
    add("4b go test ./...", FLAG, "超时（>300s），请手动运行")

# ---------- 总结 ----------
n_pass = sum(1 for _, s, _ in results if s == PASS)
n_fail = sum(1 for _, s, _ in results if s == FAIL)
n_flag = sum(1 for _, s, _ in results if s == FLAG)
print("=" * 60)
print(f"总结：PASS {n_pass} / FAIL {n_fail} / RED FLAG {n_flag}")
if n_fail == 0 and n_flag == 0:
    print("代码层全部通过。请继续做文档中的 UI 手工测试（能不能真写、真查、不编造）。")
else:
    print("代码层未达标。把上面 FAIL / RED FLAG 的条目原样发给 Codex 要求修正。")
