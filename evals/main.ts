import "dotenv/config";
import { ToolLoopAgent, tool, generateText, stepCountIs } from "ai";
import { openai } from "@ai-sdk/openai";
import { z } from "zod";
import { execSync } from "child_process";
import { readFileSync, readdirSync, writeFileSync } from "fs";
import { join } from "path";

// --- Types ---

interface Scenario {
  name: string;
  category: string;
  description: string;
  user_intent: string;
  expected_outcome: string;
  max_turns: number;
}

interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

interface TurnResult {
  turn: number;
  command?: string;
  exit_code?: number;
  output?: string;
  done: boolean;
  summary?: string;
}

interface VerificationResult {
  resources_created: string[];
  commands_succeeded: number;
  commands_failed: number;
  has_real_output: boolean;
}

interface ScenarioResult {
  name: string;
  category: string;
  user_intent: string;
  expected_outcome: string;
  mode: string;
  turns: TurnResult[];
  total_turns: number;
  score: number;
  verdict: string;
  reasoning: string;
  issues: string[];
  duration_ms: number;
  agent_tokens: TokenUsage;
  judge_tokens: TokenUsage;
  verification: VerificationResult;
}

interface JudgeResponse {
  score: number;
  verdict: string;
  reasoning: string;
  issues: string[];
}

// --- Config ---

const RUNPOD_API_KEY = process.env.RUNPOD_API_KEY;
const MODEL = "gpt-5.2";

if (!RUNPOD_API_KEY) {
  console.error("RUNPOD_API_KEY must be set");
  process.exit(1);
}
// OPENAI_API_KEY is read automatically by @ai-sdk/openai

// --- Helpers ---

function findBinary(): string {
  try {
    const stat = require("fs").statSync("dist/rpcli");
    if (stat) return join(process.cwd(), "dist/rpcli");
  } catch {}
  return "go run ./cmd/rpcli";
}

function executeCLI(binary: string, args: string[], truncate = true): { output: string; exitCode: number } {
  let cmd: string;
  if (binary.includes("go run")) {
    cmd = `go run ./cmd/rpcli --api-key ${RUNPOD_API_KEY} ${args.join(" ")}`;
  } else {
    cmd = `${binary} --api-key ${RUNPOD_API_KEY} ${args.join(" ")}`;
  }

  try {
    const stdout = execSync(cmd, { timeout: 30_000, encoding: "utf-8", stdio: ["pipe", "pipe", "pipe"] });
    let output = stdout.toString();
    if (truncate && output.length > 3000) output = output.slice(0, 3000) + "\n... (truncated)";
    return { output, exitCode: 0 };
  } catch (err: any) {
    const output = (err.stdout?.toString() || err.stderr?.toString() || err.message).slice(0, 3000);
    return { output, exitCode: err.status ?? 1 };
  }
}

function loadScenarios(dir: string): Scenario[] {
  const files = readdirSync(dir)
    .filter((f) => f.endsWith(".json"))
    .sort();
  return files.map((f) => JSON.parse(readFileSync(join(dir, f), "utf-8")));
}

function truncate(s: string, n: number): string {
  return s.length <= n ? s : s.slice(0, n) + "...";
}

function colorScore(score: number): string {
  if (score >= 7) return `\x1b[32m${score}/10\x1b[0m`;
  if (score >= 5) return `\x1b[33m${score}/10\x1b[0m`;
  return `\x1b[31m${score}/10\x1b[0m`;
}

// --- Agent ---

async function runAgentChat(
  binary: string,
  skillMD: string,
  scenario: Scenario,
): Promise<{ turns: TurnResult[]; tokens: TokenUsage }> {
  const maxSteps = (scenario.max_turns || 10) * 2; // each "turn" = tool call + response

  let skillSection = "";
  if (skillMD) {
    skillSection = `\n\n## CLI Documentation\n${skillMD}`;
  }

  const instructions = `You are an AI agent that manages GPU cloud infrastructure using a CLI tool called "rpcli".
You can execute rpcli commands to accomplish tasks. You have access to a Runpod account via API key (already configured).
${skillSection}

Rules:
- Use one command at a time to accomplish the task
- Use --yes for destructive operations (stop, delete, restart, reset)
- Parse JSON output from previous commands to extract IDs and other values
- If a command fails, analyze the error and try a different approach
- You can discover available commands by running "--help" or "<subcommand> --help"
- When you are done, stop calling tools and provide a final summary`;

  const turns: TurnResult[] = [];
  let turnNum = 0;

  const agent = new ToolLoopAgent({
    model: openai(MODEL),
    instructions,
    tools: {
      execute_cli: tool({
        description: "Execute an rpcli command. Pass the arguments as a single string (without the 'rpcli' prefix).",
        inputSchema: z.object({
          args: z.string().describe('The rpcli command arguments, e.g. "pod list" or "resource gpu"'),
        }),
        execute: async ({ args }) => {
          turnNum++;
          const argParts = args.split(/\s+/);
          const { output, exitCode } = executeCLI(binary, argParts);

          console.log(`    Turn ${turnNum}: rpcli ${truncate(args, 120)}`);

          turns.push({
            turn: turnNum,
            command: `rpcli ${args}`,
            exit_code: exitCode,
            output,
            done: false,
          });

          return { exit_code: exitCode, output };
        },
      }),
    },
    stopWhen: stepCountIs(maxSteps),
  });

  let agentTokens: TokenUsage = { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 };

  const result = await agent.generate({
    prompt: scenario.user_intent,
    onFinish({ totalUsage }) {
      agentTokens = {
        prompt_tokens: totalUsage.inputTokens ?? 0,
        completion_tokens: totalUsage.outputTokens ?? 0,
        total_tokens: totalUsage.totalTokens ?? 0,
      };
    },
  });

  // Record the final text response as the done turn
  if (result.text) {
    turnNum++;
    turns.push({
      turn: turnNum,
      done: true,
      summary: result.text,
    });
    console.log(`    Turn ${turnNum}: DONE — ${truncate(result.text, 100)}`);
  }

  return { turns, tokens: agentTokens };
}

// --- Judge ---

async function judgeConversation(
  scenario: Scenario,
  turns: TurnResult[],
  withSkill: boolean,
  verification: VerificationResult,
): Promise<{ verdict: JudgeResponse; tokens: TokenUsage }> {
  const skillNote = withSkill
    ? "The agent had SKILL.md documentation available."
    : "The agent had NO tool documentation — it had to discover commands on its own.";

  const prompt = `You are judging an AI agent's ability to use a CLI tool called "rpcli" to manage Runpod GPU cloud infrastructure.

## Context
${skillNote}

## Scenario
Name: ${scenario.name}
Category: ${scenario.category}
Description: ${scenario.description}
User Intent: ${scenario.user_intent}
Expected Outcome: ${scenario.expected_outcome}

## Agent Conversation Transcript
${JSON.stringify(turns, null, 2)}

## Programmatic Verification (ground truth)
These are the ACTUAL resources that exist on the Runpod account after the agent ran:
- Resources created: ${verification.resources_created.length > 0 ? verification.resources_created.join(", ") : "none"}
- Commands succeeded: ${verification.commands_succeeded}
- Commands failed: ${verification.commands_failed}
- Produced real output: ${verification.has_real_output}

This verification is ground truth. If the expected outcome requires creating a resource and it does NOT appear in the verification, the task FAILED regardless of what the transcript shows.

## Evaluation Criteria (score 1-10)
1. **Task Completion** (4pts): Did the agent achieve the expected outcome? Cross-reference with the verification — did resources actually get created?
2. **Efficiency** (2pts): Did the agent accomplish the task in a reasonable number of turns?
3. **Error Recovery** (2pts): If commands failed, did the agent adapt?
4. **Understanding** (2pts): Did the agent demonstrate understanding of the CLI and task domain?

Score 7+ = pass. An agent that fails to figure out the right commands or gives up early should score low.
If the verification shows no resources were created but the task required creating resources, score <= 4 — UNLESS the expected outcome explicitly allows for a platform-level failure (e.g. "no instances available" for spot pods). In that case, judge whether the agent used the correct commands and handled the error gracefully.

Respond with ONLY valid JSON (no markdown fences):
{"score": <1-10>, "verdict": "pass|warn|fail", "reasoning": "<1-2 sentences>", "issues": ["issue1", "issue2"]}`;

  let judgeTokens: TokenUsage = { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 };

  try {
    const result = await generateText({
      model: openai(MODEL),
      prompt,
      maxOutputTokens: 500,
      onFinish({ usage }) {
        judgeTokens = {
          prompt_tokens: usage.inputTokens ?? 0,
          completion_tokens: usage.outputTokens ?? 0,
          total_tokens: usage.totalTokens ?? 0,
        };
      },
    });

    let content = result.text.trim();
    content = content.replace(/^```json\s*/, "").replace(/```$/, "").trim();

    const judge: JudgeResponse = JSON.parse(content);
    return { verdict: judge, tokens: judgeTokens };
  } catch (err: any) {
    return {
      verdict: { score: 0, verdict: "fail", reasoning: `Judge error: ${err.message}`, issues: [] },
      tokens: judgeTokens,
    };
  }
}

// --- Verification ---

const EVAL_PREFIX = "rpcli-eval";

function isEvalResource(name: string): boolean {
  const lower = name.toLowerCase();
  return lower.includes("rpcli-eval") || lower.includes("rpcli_eval");
}

function verifyResources(binary: string, turns: TurnResult[]): VerificationResult {
  const tryParse = (json: string) => {
    try { return JSON.parse(json); } catch { return []; }
  };

  // Count command outcomes from the transcript
  let succeeded = 0, failed = 0;
  for (const t of turns) {
    if (t.command) {
      if (t.exit_code === 0) succeeded++;
      else failed++;
    }
  }

  // Check what rpcli-eval-* resources actually exist right now
  const resources: string[] = [];

  // Use truncate=false so JSON isn't broken by the 3000-char limit
  const pods = tryParse(executeCLI(binary, ["pod", "list"], false).output) as any[];
  for (const p of pods) {
    if (isEvalResource(p.name || "")) resources.push(`pod:${p.name}`);
  }

  const endpoints = tryParse(executeCLI(binary, ["endpoint", "list"], false).output) as any[];
  for (const e of endpoints) {
    if (isEvalResource(e.name || "")) resources.push(`endpoint:${e.name}`);
  }

  const templates = tryParse(executeCLI(binary, ["template", "list"], false).output) as any[];
  for (const t of templates) {
    if (isEvalResource(t.name || "") || (t.name || "").startsWith("rpcli-ep-")) resources.push(`template:${t.name}`);
  }

  const secrets = tryParse(executeCLI(binary, ["secret", "list"], false).output) as any[];
  for (const s of secrets) {
    if (isEvalResource(s.name || "")) resources.push(`secret:${s.name}`);
  }

  const registries = tryParse(executeCLI(binary, ["registry", "list"], false).output) as any[];
  for (const r of registries) {
    if (isEvalResource(r.name || "")) resources.push(`registry:${r.name}`);
  }

  const volumes = tryParse(executeCLI(binary, ["volume", "list"], false).output) as any[];
  for (const v of volumes) {
    if (isEvalResource(v.name || "")) resources.push(`volume:${v.name}`);
  }

  // Check if any turn produced real output (not just errors)
  // An empty list [] is valid real output (e.g. no volumes exist)
  const hasRealOutput = turns.some(
    (t) => t.exit_code === 0 && t.output !== undefined && !t.output.includes('"error"'),
  );

  return {
    resources_created: resources,
    commands_succeeded: succeeded,
    commands_failed: failed,
    has_real_output: hasRealOutput,
  };
}

// --- Cleanup ---

function cleanupEvalResources(binary: string) {
  const tryParse = (json: string) => {
    try { return JSON.parse(json); } catch { return []; }
  };

  // Pods: stop then delete (truncate=false to avoid broken JSON)
  const pods = tryParse(executeCLI(binary, ["pod", "list"], false).output) as any[];
  for (const pod of pods) {
    if (isEvalResource(pod.name || "")) {
      console.log(`  Stopping pod: ${pod.name} (${pod.id})`);
      executeCLI(binary, ["pod", "stop", pod.id, "--yes"]);
      console.log(`  Deleting pod: ${pod.name} (${pod.id})`);
      executeCLI(binary, ["pod", "delete", pod.id, "--yes"]);
    }
  }

  // Endpoints
  const endpoints = tryParse(executeCLI(binary, ["endpoint", "list"], false).output) as any[];
  for (const ep of endpoints) {
    if (isEvalResource(ep.name || "")) {
      console.log(`  Deleting endpoint: ${ep.name} (${ep.id})`);
      executeCLI(binary, ["endpoint", "delete", ep.id, "--yes"]);
    }
  }

  // Templates (includes auto-created rpcli-ep-* templates from endpoints)
  const templates = tryParse(executeCLI(binary, ["template", "list"], false).output) as any[];
  for (const t of templates) {
    if (isEvalResource(t.name || "") || (t.name || "").startsWith("rpcli-ep-")) {
      console.log(`  Deleting template: ${t.name}`);
      executeCLI(binary, ["template", "delete", t.name, "--yes"]);
    }
  }

  // Secrets
  const secrets = tryParse(executeCLI(binary, ["secret", "list"], false).output) as any[];
  for (const s of secrets) {
    if (isEvalResource(s.name || "")) {
      console.log(`  Deleting secret: ${s.name} (${s.id})`);
      executeCLI(binary, ["secret", "delete", s.id, "--yes"]);
    }
  }

  // Registries
  const registries = tryParse(executeCLI(binary, ["registry", "list"], false).output) as any[];
  for (const r of registries) {
    if (isEvalResource(r.name || "")) {
      console.log(`  Deleting registry: ${r.name} (${r.id})`);
      executeCLI(binary, ["registry", "delete", r.id, "--yes"]);
    }
  }

  // Volumes
  const volumes = tryParse(executeCLI(binary, ["volume", "list"], false).output) as any[];
  for (const v of volumes) {
    if (isEvalResource(v.name || "")) {
      console.log(`  Deleting volume: ${v.name} (${v.id})`);
      executeCLI(binary, ["volume", "delete", v.id, "--yes"]);
    }
  }

  console.log("  Cleanup done.");
}

// --- Main ---

async function main() {
  const binary = findBinary();
  const skillMD = (() => {
    try { return readFileSync("rpcli/SKILL.md", "utf-8"); } catch { return ""; }
  })();
  const allScenarios = loadScenarios("evals/scenarios");

  // Allow running specific scenarios by number: tsx evals/main.ts 5 7 8 16
  const filterArgs = process.argv.slice(2).map(Number).filter(n => n > 0);
  const scenarios = filterArgs.length > 0
    ? allScenarios.filter((_, i) => filterArgs.includes(i + 1))
    : allScenarios;

  console.log(`Loaded ${allScenarios.length} scenarios, running ${scenarios.length}, binary: ${binary}`);
  if (filterArgs.length > 0) console.log(`Running scenarios: ${filterArgs.join(", ")}`);
  console.log(`SKILL.md: ${skillMD.length} bytes\n`);

  // Clean slate — remove any leftover eval resources from previous runs
  console.log("=== Pre-run cleanup ===");
  cleanupEvalResources(binary);
  console.log();

  const results: ScenarioResult[] = [];
  let withSkillPass = 0, withSkillFail = 0;
  let noSkillPass = 0, noSkillFail = 0;

  for (let i = 0; i < scenarios.length; i++) {
    const sc = scenarios[i];
    const scenarioNum = filterArgs.length > 0 ? filterArgs[i] : i + 1;
    console.log(`[${scenarioNum}/${allScenarios.length}] ${sc.category} — ${sc.name}`);

    // --- WITH SKILL ---
    console.log("  Agent chat WITH SKILL.md...");
    const withStart = Date.now();
    const withResult = await runAgentChat(binary, skillMD, sc);
    const withElapsed = Date.now() - withStart;

    // Verify immediately while resources still exist
    console.log("  Verifying WITH...");
    const withVerification = verifyResources(binary, withResult.turns);
    if (withVerification.resources_created.length > 0) {
      console.log(`    Resources found: ${withVerification.resources_created.join(", ")}`);
    } else {
      console.log("    No eval resources found on account");
    }
    console.log(`    Commands: ${withVerification.commands_succeeded} ok, ${withVerification.commands_failed} failed`);

    process.stdout.write("  Judging WITH... ");
    const withJudge = await judgeConversation(sc, withResult.turns, true, withVerification);
    console.log(`WITH SKILL [${colorScore(withJudge.verdict.score)}] ${withJudge.verdict.reasoning}`);
    for (const iss of withJudge.verdict.issues) console.log(`    - ${iss}`);

    results.push({
      name: sc.name, category: sc.category, user_intent: sc.user_intent,
      expected_outcome: sc.expected_outcome, mode: "with_skill",
      turns: withResult.turns, total_turns: withResult.turns.length,
      score: withJudge.verdict.score, verdict: withJudge.verdict.verdict,
      reasoning: withJudge.verdict.reasoning, issues: withJudge.verdict.issues,
      duration_ms: withElapsed,
      agent_tokens: withResult.tokens, judge_tokens: withJudge.tokens,
      verification: withVerification,
    });
    if (withJudge.verdict.score >= 7) withSkillPass++; else withSkillFail++;

    // Clean up between WITH and WITHOUT so names don't collide
    console.log("  Cleaning up before WITHOUT run...");
    cleanupEvalResources(binary);

    // --- WITHOUT SKILL ---
    console.log("  Agent chat WITHOUT SKILL.md...");
    const noStart = Date.now();
    const noResult = await runAgentChat(binary, "", sc);
    const noElapsed = Date.now() - noStart;

    // Verify immediately while resources still exist
    console.log("  Verifying WITHOUT...");
    const noVerification = verifyResources(binary, noResult.turns);
    if (noVerification.resources_created.length > 0) {
      console.log(`    Resources found: ${noVerification.resources_created.join(", ")}`);
    } else {
      console.log("    No eval resources found on account");
    }
    console.log(`    Commands: ${noVerification.commands_succeeded} ok, ${noVerification.commands_failed} failed`);

    process.stdout.write("  Judging WITHOUT... ");
    const noJudge = await judgeConversation(sc, noResult.turns, false, noVerification);
    console.log(`NO   SKILL [${colorScore(noJudge.verdict.score)}] ${noJudge.verdict.reasoning}`);
    for (const iss of noJudge.verdict.issues) console.log(`    - ${iss}`);

    results.push({
      name: sc.name, category: sc.category, user_intent: sc.user_intent,
      expected_outcome: sc.expected_outcome, mode: "without_skill",
      turns: noResult.turns, total_turns: noResult.turns.length,
      score: noJudge.verdict.score, verdict: noJudge.verdict.verdict,
      reasoning: noJudge.verdict.reasoning, issues: noJudge.verdict.issues,
      duration_ms: noElapsed,
      agent_tokens: noResult.tokens, judge_tokens: noJudge.tokens,
      verification: noVerification,
    });
    if (noJudge.verdict.score >= 7) noSkillPass++; else noSkillFail++;

    // Clean up between scenarios
    console.log("  Cleaning up after scenario...");
    cleanupEvalResources(binary);

    console.log();
  }

  // Summary
  const total = scenarios.length;
  console.log("\n" + "=".repeat(70));
  console.log(`WITH SKILL.md:    ${withSkillPass}/${total} passed (${withSkillFail} failed)`);
  console.log(`WITHOUT SKILL.md: ${noSkillPass}/${total} passed (${noSkillFail} failed)`);

  let avgWith = 0, avgNo = 0;
  for (const r of results) {
    if (r.mode === "with_skill") avgWith += r.score;
    else avgNo += r.score;
  }
  if (total > 0) { avgWith /= total; avgNo /= total; }
  console.log(`Average score WITH:    ${avgWith.toFixed(1)}/10`);
  console.log(`Average score WITHOUT: ${avgNo.toFixed(1)}/10`);
  console.log(`Skill improvement:     ${(avgWith - avgNo) >= 0 ? "+" : ""}${(avgWith - avgNo).toFixed(1)} points`);

  // Token summary
  const withAgent = { p: 0, c: 0, t: 0 };
  const withJudgeT = { p: 0, c: 0, t: 0 };
  const noAgent = { p: 0, c: 0, t: 0 };
  const noJudgeT = { p: 0, c: 0, t: 0 };
  for (const r of results) {
    const a = r.mode === "with_skill" ? withAgent : noAgent;
    const j = r.mode === "with_skill" ? withJudgeT : noJudgeT;
    a.p += r.agent_tokens.prompt_tokens; a.c += r.agent_tokens.completion_tokens; a.t += r.agent_tokens.total_tokens;
    j.p += r.judge_tokens.prompt_tokens; j.c += r.judge_tokens.completion_tokens; j.t += r.judge_tokens.total_tokens;
  }
  console.log("\nToken Usage:");
  console.log(`  WITH SKILL    — agent: ${withAgent.t} tokens (prompt: ${withAgent.p}, completion: ${withAgent.c}) | judge: ${withJudgeT.t} tokens`);
  console.log(`  WITHOUT SKILL — agent: ${noAgent.t} tokens (prompt: ${noAgent.p}, completion: ${noAgent.c}) | judge: ${noJudgeT.t} tokens`);
  console.log(`  Skill token overhead: ${withAgent.t - noAgent.t >= 0 ? "+" : ""}${withAgent.t - noAgent.t} agent tokens`);

  // Verification summary
  let withCreated = 0, noCreated = 0;
  let withCmdOk = 0, withCmdFail = 0, noCmdOk = 0, noCmdFail = 0;
  for (const r of results) {
    if (r.mode === "with_skill") {
      withCreated += r.verification.resources_created.length;
      withCmdOk += r.verification.commands_succeeded;
      withCmdFail += r.verification.commands_failed;
    } else {
      noCreated += r.verification.resources_created.length;
      noCmdOk += r.verification.commands_succeeded;
      noCmdFail += r.verification.commands_failed;
    }
  }
  console.log("\nVerification:");
  console.log(`  WITH SKILL    — resources created: ${withCreated} | commands: ${withCmdOk} ok, ${withCmdFail} failed`);
  console.log(`  WITHOUT SKILL — resources created: ${noCreated} | commands: ${noCmdOk} ok, ${noCmdFail} failed`);

  // Write report
  writeFileSync("evals/report.json", JSON.stringify(results, null, 2));
  console.log("\nFull report: evals/report.json");

  // Shortcomings
  const shortcomings = results
    .filter((r) => r.score < 7 && r.mode === "with_skill")
    .map((r) => {
      let entry = `## ${r.name} (score: ${r.score}/10)\n**Category:** ${r.category}\n**Reasoning:** ${r.reasoning}\n`;
      for (const iss of r.issues) entry += `- ${iss}\n`;
      return entry;
    });
  if (shortcomings.length > 0) {
    const report = "# rpcli Eval Shortcomings Report\n\nThese scenarios scored below 7/10 with SKILL.md context.\n\n" + shortcomings.join("\n");
    writeFileSync("evals/SHORTCOMINGS.md", report);
    console.log("Shortcomings: evals/SHORTCOMINGS.md");
  } else {
    console.log("No shortcomings found (all scenarios >= 7/10 with skill).");
  }
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
