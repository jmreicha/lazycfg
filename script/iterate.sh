#!/usr/bin/env bash
#
# Iterate - Autonomous coding loop for beads
#
# Runs an AI agent in a loop to complete tasks from beads issues.
#
# Usage: ./script/iterate.sh [--max-iterations N] [--issue-id ID] [--model MODEL] [--timeout SECONDS]

set -euo pipefail

# Configuration
MAX_ITERATIONS="${MAX_ITERATIONS:-3}"
AI_AGENT="${AI_AGENT:-opencode run}"    # Default to opencode, override via env
STREAM_OUTPUT="${STREAM_OUTPUT:-false}" # Set to true to stream AI output in real-time
PROGRESS_FILE="context/progress.txt"
PROMPT_FILE="context/prompt.md"
FINDINGS_MARKER="<findings>"
FINDINGS_END_MARKER="</findings>"
COMPLETION_MARKER="<promise>COMPLETE</promise>"
AGENTS_MD="AGENTS.md"
MAX_ATTEMPTS_PER_ISSUE=3
OPENCODE_TIMEOUT="${OPENCODE_TIMEOUT:-900}" # Default 15 mins, override via env
OPENCODE_MODEL="${OPENCODE_MODEL:-github-copilot/claude-sonnet-4.5}"
declare -A issue_attempts         # Track retry count per issue ID
declare -a successful_attempts=() # Track attempts for completed issues (for avg calculation)
issues_completed=0
issues_failed=0
current_pid=""

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
    --model)
        OPENCODE_MODEL="$2"
        shift 2
        ;;
    --timeout)
        OPENCODE_TIMEOUT="$2"
        shift 2
        ;;
    -h | --help)

        cat <<HELP
Iterate - Autonomous coding loop for beads

Usage: $0 [OPTIONS]

Options:
  --max-iterations N    Maximum iterations (default: 3)
  --issue-id ID        Work on specific issue
  --model MODEL        OpenCode model (default: github-copilot/claude-sonnet-4.5)
  --timeout SECONDS    OpenCode timeout in seconds (default: 900)
  -h, --help           Show this help

Environment Variables:
  MAX_ITERATIONS       Override default max iterations
  AI_AGENT             AI agent command (default: "opencode run")
  OPENCODE_MODEL       OpenCode model (default: github-copilot/claude-sonnet-4.5)
  OPENCODE_TIMEOUT     Timeout in seconds (default: 900)
  STREAM_OUTPUT        Set to true to stream AI output in real-time (default: false)

Examples:
  $0                                                   # Run with defaults (opencode)
  $0 --max-iterations 100                              # Custom max
  $0 --issue-id abc123                                 # Specific issue
  $0 --model github-copilot/claude-sonnet-4.5          # Set model
  $0 --timeout 3600                                    # 1 hour timeout
  MAX_ITERATIONS=25 $0                                 # Override with env
  AI_AGENT="claude --dangerously-skip-permissions" $0  # Use Claude CLI
  OPENCODE_TIMEOUT=3600 $0                             # 1 hour timeout
  STREAM_OUTPUT=true $0                                # Enable output streaming

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

# Validate required environment variables
if [[ -z "${GITHUB_TOKEN:-}" ]]; then
    echo -e "${RED}Error: GITHUB_TOKEN environment variable is not set${NC}"
    echo "Please set GITHUB_TOKEN and try again."
    exit 1
fi

# Ensure progress file exists
mkdir -p context
if [[ ! -f "$PROGRESS_FILE" ]]; then
    cat >"$PROGRESS_FILE" <<INIT
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
    bd ready --json 2>/dev/null | jq -r '.[0].id // empty' || echo ""
}

# Append findings to AGENTS.md
append_finding() {
    local finding="$1"
    if grep -q "$finding" "$AGENTS_MD" 2>/dev/null; then
        return
    fi
    echo "- $finding" >>"$AGENTS_MD"
    echo -e "${GREEN}📝 Recorded finding${NC}"
}

# Write session improvements to progress.txt
record_improvements() {
    local improvements="$1"

    cat >>"$PROGRESS_FILE" <<IMPROV

### $(date '+%Y-%m-%d %H:%M:%S')

$improvements

---
IMPROV
}

# Invoke OpenCode autonomously for current issue
invoke_opencode() {
    local issue_id="$1"
    local output_file
    output_file=$(mktemp)

    # Create debug log file in /tmp
    local debug_log="/tmp/opencode-${issue_id}-$(date +%Y%m%d-%H%M%S).log"

    # Build prompt: issue details + instructions + recent learnings
    local prompt
    prompt=$(
        cat <<PROMPT
# Beads Issue to Complete

$(bd show "$issue_id" 2>/dev/null || echo "Issue: $issue_id")

---

# Instructions

$(cat "$PROMPT_FILE" 2>/dev/null || echo "")

---

# Recent Learnings (Last 30 lines)

$(tail -30 "$PROGRESS_FILE" 2>/dev/null || echo "No history yet")

PROMPT
    )

    echo -e "\n${YELLOW}🚀 Invoking AI agent (timeout: ${OPENCODE_TIMEOUT}s)...${NC}"
    echo -e "${BLUE}Agent: ${AI_AGENT}${NC}"
    echo -e "${BLUE}Model: ${OPENCODE_MODEL}${NC}"

    echo -e "${BLUE}Debug log: $debug_log${NC}\n"

    # Invoke AI agent with timeout and capture output
    set +e # Don't exit on error

    # Stream output if enabled, otherwise capture silently
    # Always save to debug log in /tmp (streamable for tail -f)
    if [[ "$STREAM_OUTPUT" == "true" ]]; then
        echo -e "${BLUE}Streaming AI agent output...${NC}\n"
        timeout "$OPENCODE_TIMEOUT" $AI_AGENT "$prompt" --model "$OPENCODE_MODEL" 2>&1 | tee "$output_file" "$debug_log"
    else
        timeout "$OPENCODE_TIMEOUT" $AI_AGENT "$prompt" --model "$OPENCODE_MODEL" 2>&1 | tee "$output_file" "$debug_log" >/dev/null

    fi

    current_pid="${!}"
    local exit_code="${PIPESTATUS[0]}"
    current_pid=""
    set -e

    local output
    output=$(cat "$output_file")

    rm -f "$output_file"

    # Check for timeout
    if [[ $exit_code -eq 124 ]]; then
        echo -e "\n${RED}⚠ AI agent timed out after ${OPENCODE_TIMEOUT} seconds ($(($OPENCODE_TIMEOUT / 60)) minutes)${NC}"
        return 1
    fi

    # Check exit code
    if [[ $exit_code -ne 0 ]]; then
        echo -e "\n${RED}⚠ AI agent exited with code $exit_code${NC}"
        return 1
    fi

    # Check for completion marker
    if echo "$output" | grep -q "$COMPLETION_MARKER"; then
        echo -e "\n${GREEN}✓ Completion marker found${NC}"

        # Extract and save findings
        local findings
        findings=$(echo "$output" | sed -n '/<findings>/,/<\/findings>/p' | sed '/<findings>/d;/<\/findings>/d')

        if [[ -n "$findings" ]]; then
            echo -e "${BLUE}📝 Extracting findings...${NC}"
            # Append findings to AGENTS.md (one per line)
            while IFS= read -r finding; do
                [[ -n "$finding" ]] && append_finding "$finding"
            done <<<"$findings"
        fi

        # Mark issue complete
        echo -e "${GREEN}✓ Marking issue complete${NC}"
        bd update "$issue_id" --status done 2>/dev/null || true

        return 0
    else
        echo -e "\n${YELLOW}⚠ Completion marker NOT found${NC}"
        return 1
    fi
}

# Main loop
iteration=1
current_issue=""
start_time=$(date +%s)
stop_requested=false

handle_sigint() {
    stop_requested=true
    if [[ -n "$current_pid" ]]; then
        kill -INT "$current_pid" 2>/dev/null || true
    fi
    echo -e "\n${YELLOW}Stop requested. Finishing current iteration...${NC}"
}

trap handle_sigint INT

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
    echo ""

    # Initialize attempt counter if not set
    if [[ -z "${issue_attempts[$current_issue]:-}" ]]; then
        issue_attempts[$current_issue]=0
    fi

    # Increment attempt counter
    issue_attempts[$current_issue]=$((${issue_attempts[$current_issue]} + 1))
    attempts="${issue_attempts[$current_issue]}"

    echo -e "${YELLOW}Attempt $attempts/$MAX_ATTEMPTS_PER_ISSUE for $current_issue${NC}"

    # Invoke OpenCode
    if invoke_opencode "$current_issue"; then
        # Success - move to next issue
        echo -e "${GREEN}✓ Issue $current_issue completed successfully${NC}"

        # Track metrics
        successful_attempts+=("$attempts")
        issues_completed=$((issues_completed + 1))

        # Clear retry counter
        unset "issue_attempts[$current_issue]"
        current_issue=""
    else
        # Failed - check retry limit
        if [[ $attempts -ge $MAX_ATTEMPTS_PER_ISSUE ]]; then
            echo -e "${RED}✗ Max attempts ($MAX_ATTEMPTS_PER_ISSUE) reached for $current_issue${NC}"

            # Track metrics
            issues_failed=$((issues_failed + 1))

            echo -e "${YELLOW}Moving to next issue (marked $current_issue as blocked)${NC}"
            unset "issue_attempts[$current_issue]"
            current_issue=""
        else
            echo -e "${YELLOW}Will retry $current_issue (attempt $((attempts + 1))/$MAX_ATTEMPTS_PER_ISSUE)${NC}"
            # Don't clear current_issue - will retry next iteration
        fi
    fi

    iteration=$((iteration + 1))

    if [[ "$stop_requested" == "true" ]]; then
        echo -e "${YELLOW}Stop requested. Exiting loop.${NC}"
        break
    fi

done

# Session summary
end_time=$(date +%s)
duration=$((end_time - start_time))
duration_formatted=$(printf '%02d:%02d:%02d' $((duration / 3600)) $((duration % 3600 / 60)) $((duration % 60)))

# Calculate average attempts for successful issues
avg_attempts="N/A"
if [[ ${#successful_attempts[@]} -gt 0 ]]; then
    sum=0
    for attempt in "${successful_attempts[@]}"; do
        sum=$((sum + attempt))
    done
    count=${#successful_attempts[@]}
    avg_attempts=$(awk "BEGIN {printf \"%.1f\", $sum / $count}")
fi

echo -e "\n${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Loop Complete${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "Iterations: $((iteration - 1))"
echo -e "Duration: $duration_formatted"
echo -e "\n${BLUE}Issue Statistics:${NC}"
echo -e "  Completed: ${GREEN}$issues_completed${NC}"
echo -e "  Failed: ${RED}$issues_failed${NC}"
echo -e "  Avg attempts (successful): $avg_attempts"

echo -e "${GREEN}Done!${NC}"
