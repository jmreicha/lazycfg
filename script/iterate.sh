#!/usr/bin/env bash
#
# Iterate - Autonomous coding loop for beads
#
# Runs an AI agent in a loop to complete tasks from beads issues.
#
# Usage: ./script/iterate.sh [--max-iterations N] [--issue-id ID]

set -euo pipefail

# Configuration
MAX_ITERATIONS="${MAX_ITERATIONS:-50}"
PROGRESS_FILE="context/progress.txt"
PROMPT_FILE="context/prompt.md"
FINDINGS_MARKER="<findings>"
FINDINGS_END_MARKER="</findings>"
COMPLETION_MARKER="<promise>COMPLETE</promise>"
AGENTS_MD="AGENTS.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
ISSUE_ID=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --max-iterations)
      MAX_ITERATIONS="$2"
      shift 2
      ;;
    --issue-id)
      ISSUE_ID="$2"
      shift 2
      ;;
    -h|--help)
      cat <<HELP
Iterate - Autonomous coding loop for beads

Usage: $0 [OPTIONS]

Options:
  --max-iterations N    Maximum iterations (default: 50)
  --issue-id ID        Work on specific issue
  -h, --help           Show this help

Environment Variables:
  MAX_ITERATIONS       Override default max iterations

Examples:
  $0                           # Run with defaults
  $0 --max-iterations 100      # Custom max
  $0 --issue-id abc123         # Specific issue
  MAX_ITERATIONS=25 $0         # Override with env

Runs an AI agent in a loop to autonomously complete beads issues.
HELP
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Try '$0 --help' for more information."
      exit 1
      ;;
  esac
done

# Ensure progress file exists
mkdir -p context
if [[ ! -f "$PROGRESS_FILE" ]]; then
  cat > "$PROGRESS_FILE" <<INIT
# Autonomous Loop Progress

This file tracks learnings from autonomous work sessions.

## Format
Each entry includes improvements and learnings from each session.

---

INIT
fi

echo -e "${BLUE}🤖 Starting autonomous loop${NC}"
echo -e "${BLUE}Max iterations: $MAX_ITERATIONS${NC}"

# Get next ready issue
get_next_issue() {
  if [[ -n "$ISSUE_ID" ]]; then
    echo "$ISSUE_ID"
    return
  fi
  bd ready --format json 2>/dev/null | jq -r '.[0].id // empty' || echo ""
}

# Append findings to AGENTS.md
append_finding() {
  local finding="$1"
  if grep -q "$finding" "$AGENTS_MD" 2>/dev/null; then
    return
  fi
  echo "- $finding" >> "$AGENTS_MD"
  echo -e "${GREEN}📝 Recorded finding${NC}"
}

# Write session improvements to progress.txt
record_improvements() {
  local improvements="$1"

  cat >> "$PROGRESS_FILE" <<IMPROV

### $(date '+%Y-%m-%d %H:%M:%S')

$improvements

---
IMPROV
}

# Main loop
iteration=1
current_issue=""
start_time=$(date +%s)

while [[ $iteration -le $MAX_ITERATIONS ]]; do
  echo -e "\n${YELLOW}━━━ Iteration $iteration/$MAX_ITERATIONS ━━━${NC}\n"

  # Get issue
  if [[ -z "$current_issue" ]]; then
    current_issue=$(get_next_issue)
    if [[ -z "$current_issue" ]]; then
      echo -e "${GREEN}✓ No more issues${NC}"
      break
    fi
    echo -e "${BLUE}Issue: $current_issue${NC}"
  fi

  # Show issue details
  echo -e "\n${BLUE}═══════════════════════════════════════${NC}"
  echo -e "${BLUE}Issue Details:${NC}"
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  bd show "$current_issue" 2>/dev/null || echo "Issue not found"

  # Show instructions
  echo -e "\n${BLUE}═══════════════════════════════════════${NC}"
  echo -e "${BLUE}Instructions:${NC}"
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  cat "$PROMPT_FILE" 2>/dev/null || echo "No prompt file found"

  # Show recent learnings
  echo -e "\n${BLUE}═══════════════════════════════════════${NC}"
  echo -e "${BLUE}Recent Learnings:${NC}"
  echo -e "${BLUE}═══════════════════════════════════════${NC}"
  tail -30 "$PROGRESS_FILE" 2>/dev/null || echo "No history yet"

  # Agent works here
  echo -e "\n${YELLOW}Press Enter when work is complete (or Ctrl+C to stop)...${NC}"
  read -r

  # Check for completion
  echo -e "Did you complete this issue? (y/n)"
  read -r completed

  if [[ "$completed" == "y" ]]; then
    echo -e "${GREEN}✓ Marking issue complete${NC}"
    bd update "$current_issue" --status done 2>/dev/null || true
    current_issue=""
  fi

  ((iteration++))
done

# Session summary
end_time=$(date +%s)
duration=$((end_time - start_time))
duration_formatted=$(printf '%02d:%02d:%02d' $((duration/3600)) $((duration%3600/60)) $((duration%60)))

echo -e "\n${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Loop Complete${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "Iterations: $((iteration - 1))"
echo -e "Duration: $duration_formatted"

# Record improvements
echo -e "\n${BLUE}Record learnings (optional):${NC}"
echo -e "What improvements for next time? (press Enter to skip)"
read -r improvements

if [[ -n "$improvements" ]]; then
  record_improvements "$improvements"
  echo -e "${GREEN}✓ Improvements recorded${NC}"
fi

# Sync beads
echo -e "\n${BLUE}Syncing beads...${NC}"
bd sync

echo -e "${GREEN}Done!${NC}"
