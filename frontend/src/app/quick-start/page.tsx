"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import yaml from "js-yaml";
import type { Agent, Environment } from "@/lib/types";
import {
  listAgents,
  listEnvironments,
  createAgent,
  createEnvironment,
  createSession,
  sendSessionEvents,
} from "@/lib/api";
import { truncateId } from "@/lib/utils";

/* ── Types ───────────────────────────────────────────────────────────── */

type BubbleRole = "bot" | "user";

interface ChatBubble {
  readonly id: string;
  readonly role: BubbleRole;
  readonly content: React.ReactNode;
}

type FlowPhase =
  | "init"
  | "pick-agent"
  | "edit-agent"
  | "pick-env"
  | "edit-env"
  | "ready"
  | "active";

type ConfigFormat = "yaml" | "json";

/* ── Templates ───────────────────────────────────────────────────────── */

const AGENT_TEMPLATE = {
  name: "Untitled agent",
  description: "A blank starting point with the core toolset.",
  model: "claude-sonnet-4-6",
  system:
    "You are a general-purpose agent that can research, write code, run commands, and use connected tools to complete the user's task end to end.",
  mcp_servers: [],
  tools: [{ type: "agent_toolset_20260401" }],
  skills: [],
};

type NetworkingType = "unrestricted" | "limited";

const TYPING_DELAY = 800;
const WIDGET_DELAY = 400;

/* ── Tiny helpers ────────────────────────────────────────────────────── */

let _seq = 0;
function nextId() {
  _seq += 1;
  return `b-${_seq}`;
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

function toYaml(obj: unknown): string {
  return yaml.dump(obj, { indent: 2, lineWidth: -1, noRefs: true, quotingType: '"' });
}

function toJson(obj: unknown): string {
  return JSON.stringify(obj, null, 2);
}

function parseConfig(text: string, format: ConfigFormat): unknown {
  return format === "yaml" ? yaml.load(text) : JSON.parse(text);
}

/* ── Small components ────────────────────────────────────────────────── */

function BotAvatar() {
  return (
    <div className="w-7 h-7 rounded-full bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0">
      <svg className="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
    </div>
  );
}

function Spinner() {
  return (
    <svg className="w-4 h-4 animate-spin text-muted-foreground" fill="none" viewBox="0 0 24 24">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
  );
}

function TypingDots() {
  return (
    <div className="flex items-start gap-2.5 animate-fadeIn">
      <BotAvatar />
      <div className="bg-muted rounded-2xl rounded-bl-md px-4 py-3 flex items-center gap-1">
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce" style={{ animationDelay: "0ms" }} />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce" style={{ animationDelay: "150ms" }} />
        <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce" style={{ animationDelay: "300ms" }} />
      </div>
    </div>
  );
}

/* ── Bubble wrapper ──────────────────────────────────────────────────── */

function Bubble({ role, children }: { readonly role: BubbleRole; readonly children: React.ReactNode }) {
  if (role === "user") {
    return (
      <div className="flex justify-end animate-fadeSlideIn">
        <div className="max-w-[75%] bg-primary text-white rounded-2xl rounded-br-md px-4 py-2.5 text-sm">
          {children}
        </div>
      </div>
    );
  }
  return (
    <div className="flex items-start gap-2.5 animate-fadeSlideIn">
      <BotAvatar />
      <div className="max-w-[75%] bg-muted rounded-2xl rounded-bl-md px-4 py-2.5 text-sm text-foreground">
        {children}
      </div>
    </div>
  );
}

/* ── Inline YAML / JSON editor ───────────────────────────────────────── */

function InlineEditor({
  value,
  onChange,
  format,
  onFormatChange,
  error,
  submitting,
  onSubmit,
  onCancel,
  submitLabel,
}: {
  readonly value: string;
  readonly onChange: (v: string) => void;
  readonly format: ConfigFormat;
  readonly onFormatChange: (f: ConfigFormat) => void;
  readonly error: string | null;
  readonly submitting: boolean;
  readonly onSubmit: () => void;
  readonly onCancel: () => void;
  readonly submitLabel: string;
}) {
  return (
    <div className="flex items-start gap-2.5 animate-fadeSlideIn">
      <BotAvatar />
      <div className="bg-muted rounded-2xl rounded-bl-md overflow-hidden max-w-[85%] w-[520px]">
        {/* Format toggle */}
        <div className="flex items-center justify-between bg-muted border-b border-border px-3 py-1.5">
          <div className="flex">
            <button
              onClick={() => onFormatChange("yaml")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                format === "yaml"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              YAML
            </button>
            <button
              onClick={() => onFormatChange("json")}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                format === "json"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              JSON
            </button>
          </div>
        </div>

        {/* Editor textarea */}
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          spellCheck={false}
          className="w-full px-4 py-3 text-sm font-mono text-foreground bg-card focus:outline-none resize-none"
          style={{ minHeight: "220px", maxHeight: "45vh" }}
        />

        {/* Error */}
        {error && (
          <div className="px-4 py-2 text-xs text-destructive bg-destructive/10 border-t border-destructive/20">
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center justify-end gap-2 px-4 py-2.5 border-t border-border">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onSubmit}
            disabled={submitting}
            className="px-4 py-1.5 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
          >
            {submitting && <Spinner />}
            {submitLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

/* ── Main page ───────────────────────────────────────────────────────── */

export default function QuickStartPage() {
  const router = useRouter();
  const bottomRef = useRef<HTMLDivElement>(null);

  const [phase, setPhase] = useState<FlowPhase>("init");
  const [bubbles, setBubbles] = useState<readonly ChatBubble[]>([]);
  const [typing, setTyping] = useState(false);
  const [widgetVisible, setWidgetVisible] = useState(false);

  // Data
  const [agents, setAgents] = useState<readonly Agent[]>([]);
  const [environments, setEnvironments] = useState<readonly Environment[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);
  const [selectedEnvId, setSelectedEnvId] = useState<string | null>(null);

  // Inline agent editor state
  const [editorText, setEditorText] = useState("");
  const [editorFormat, setEditorFormat] = useState<ConfigFormat>("yaml");
  const [editorError, setEditorError] = useState<string | null>(null);
  const [editorSubmitting, setEditorSubmitting] = useState(false);

  // Inline env form state
  const [envFormName, setEnvFormName] = useState("");
  const [envFormNetworking, setEnvFormNetworking] = useState<NetworkingType>("limited");
  const [envFormError, setEnvFormError] = useState<string | null>(null);
  const [envFormSubmitting, setEnvFormSubmitting] = useState(false);

  // Session state
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);

  /* ── Scroll helper ────────────────────────────────────────────────── */

  const scrollToBottom = useCallback(() => {
    setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }), 60);
  }, []);

  /* ── Queued bot messages with typing indicator ────────────────────── */

  const queueRef = useRef<React.ReactNode[]>([]);
  const drainingRef = useRef(false);

  const drainQueue = useCallback(async () => {
    if (drainingRef.current) return;
    drainingRef.current = true;

    while (queueRef.current.length > 0) {
      setTyping(true);
      scrollToBottom();
      await sleep(TYPING_DELAY);

      const content = queueRef.current.shift();
      setTyping(false);
      setBubbles((prev) => [...prev, { id: nextId(), role: "bot", content }]);
      scrollToBottom();

      if (queueRef.current.length > 0) {
        await sleep(300);
      }
    }

    await sleep(WIDGET_DELAY);
    setWidgetVisible(true);
    scrollToBottom();

    drainingRef.current = false;
  }, [scrollToBottom]);

  const pushBotMessages = useCallback(
    (...messages: React.ReactNode[]) => {
      setWidgetVisible(false);
      queueRef.current.push(...messages);
      drainQueue();
    },
    [drainQueue]
  );

  const pushUserBubble = useCallback(
    (content: React.ReactNode) => {
      setWidgetVisible(false);
      setBubbles((prev) => [...prev, { id: nextId(), role: "user", content }]);
      scrollToBottom();
    },
    [scrollToBottom]
  );

  /* ── Format switch helper ─────────────────────────────────────────── */

  const switchEditorFormat = useCallback(
    (newFormat: ConfigFormat) => {
      if (newFormat === editorFormat) return;
      try {
        const parsed = parseConfig(editorText, editorFormat);
        setEditorText(newFormat === "yaml" ? toYaml(parsed) : toJson(parsed));
        setEditorFormat(newFormat);
        setEditorError(null);
      } catch {
        setEditorError(
          `Cannot convert: current ${editorFormat.toUpperCase()} is invalid. Fix the syntax first.`
        );
      }
    },
    [editorText, editorFormat]
  );

  /* ── Phase 1: Init ────────────────────────────────────────────────── */

  useEffect(() => {
    let cancelled = false;
    async function init() {
      let active: Agent[] = [];
      try {
        const res = await listAgents({ limit: 100 });
        active = res.data.filter((a) => !a.archived_at);
      } catch {
        // fall through
      }
      if (cancelled) return;
      setAgents(active);
      if (active.length > 0) {
        setSelectedAgentId(active[0].id);
      }
      setPhase("pick-agent");

      pushBotMessages(
        "Hi! I'll help you start a conversation with an agent in a few quick steps.",
        active.length > 0
          ? "First, pick an agent — or create a new one."
          : "You don't have any agents yet. Let's create one first!"
      );
    }
    init();
    return () => { cancelled = true; };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  /* ── Transition helpers ───────────────────────────────────────────── */

  const transitionToEnv = useCallback(async () => {
    let active: Environment[] = [];
    try {
      const res = await listEnvironments({ limit: 100 });
      active = res.data.filter((e) => !e.archived_at);
    } catch {
      // fall through
    }
    setEnvironments(active);
    if (active.length > 0) {
      setSelectedEnvId(active[0].id);
    }
    setPhase("pick-env");

    pushBotMessages(
      active.length > 0
        ? "Great! Now pick an environment for the agent to run in."
        : "No environments found. Let's create one!"
    );
  }, [pushBotMessages]);

  const transitionToReady = useCallback(() => {
    setPhase("ready");
    pushBotMessages("All set! Type your first message below to start the session.");
  }, [pushBotMessages]);

  /* ── Agent actions ────────────────────────────────────────────────── */

  const handleConfirmAgent = async () => {
    if (!selectedAgentId) return;
    const agent = agents.find((a) => a.id === selectedAgentId);
    if (!agent) return;

    pushUserBubble(`Selected agent: ${agent.name}`);
    await transitionToEnv();
  };

  const handleOpenAgentEditor = () => {
    setWidgetVisible(false);
    setEditorText(toYaml(AGENT_TEMPLATE));
    setEditorFormat("yaml");
    setEditorError(null);
    pushBotMessages("Here's a template — edit it to your needs, then click Create.");
    setPhase("edit-agent");
  };

  const handleCancelAgentEditor = () => {
    setPhase("pick-agent");
    setWidgetVisible(true);
    scrollToBottom();
  };

  const handleSubmitAgent = async () => {
    setEditorError(null);
    setEditorSubmitting(true);
    try {
      const parsed = parseConfig(editorText, editorFormat);
      if (!parsed || typeof parsed !== "object") throw new Error("Config must be an object");
      const body = parsed as Record<string, unknown>;
      if (!body.name || typeof body.name !== "string" || !body.name.trim()) throw new Error("Agent name is required");
      if (!body.model) throw new Error("Model is required");

      const agent = await createAgent(body);
      pushUserBubble(`Created agent: ${(body.name as string).trim()}`);

      const res = await listAgents({ limit: 100 });
      const active = res.data.filter((a) => !a.archived_at);
      setAgents(active);
      setSelectedAgentId(agent.id);

      await transitionToEnv();
    } catch (err) {
      setEditorError(err instanceof Error ? err.message : "Failed to create agent");
    } finally {
      setEditorSubmitting(false);
    }
  };

  /* ── Environment actions ──────────────────────────────────────────── */

  const handleConfirmEnv = () => {
    if (!selectedEnvId) return;
    const env = environments.find((e) => e.id === selectedEnvId);
    if (!env) return;

    pushUserBubble(`Selected environment: ${env.name}`);
    transitionToReady();
  };

  const handleOpenEnvForm = () => {
    setWidgetVisible(false);
    const agentName = agents.find((a) => a.id === selectedAgentId)?.name ?? "";
    setEnvFormName(agentName ? `${agentName} Environment` : "");
    setEnvFormNetworking("limited");
    setEnvFormError(null);
    pushBotMessages("Give your environment a name and choose the network access level.");
    setPhase("edit-env");
  };

  const handleCancelEnvForm = () => {
    setPhase("pick-env");
    setWidgetVisible(true);
    scrollToBottom();
  };

  const handleSubmitEnv = async () => {
    const name = envFormName.trim();
    if (!name) {
      setEnvFormError("Name is required");
      return;
    }
    setEnvFormError(null);
    setEnvFormSubmitting(true);
    try {
      const env = await createEnvironment({
        name,
        config: {
          type: "cloud",
          networking: { type: envFormNetworking },
        },
      });
      pushUserBubble(`Created environment: ${name}`);

      const res = await listEnvironments({ limit: 100 });
      const active = res.data.filter((e) => !e.archived_at);
      setEnvironments(active);
      setSelectedEnvId(env.id);

      transitionToReady();
    } catch (err) {
      setEnvFormError(err instanceof Error ? err.message : "Failed to create environment");
    } finally {
      setEnvFormSubmitting(false);
    }
  };

  /* ── Send message ─────────────────────────────────────────────────── */

  const handleSend = async () => {
    const text = message.trim();
    if (!text || sending) return;

    pushUserBubble(text);
    setMessage("");
    setSending(true);

    try {
      if (!sessionId) {
        if (!selectedAgentId || !selectedEnvId) return;

        const session = await createSession({
          agent: selectedAgentId,
          environment_id: selectedEnvId,
        });
        setSessionId(session.id);

        await sendSessionEvents(session.id, [
          { type: "user.message", content: text },
        ]);

        setPhase("active");
        pushBotMessages(
          "Session started! Your agent is now working on it.",
          "You can continue sending messages, or view the full session detail."
        );
      } else {
        await sendSessionEvents(sessionId, [
          { type: "user.message", content: text },
        ]);
        pushBotMessages("Message sent! The agent is processing your request.");
      }
    } catch (err) {
      pushBotMessages(
        `Error: ${err instanceof Error ? err.message : "Failed to send message"}`
      );
    } finally {
      setSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  /* ── Input enabled? ───────────────────────────────────────────────── */

  const inputEnabled = phase === "ready" || phase === "active";

  /* ── Render ────────────────────────────────────────────────────────── */

  return (
    <div className="flex flex-col h-[calc(100vh-2rem)]">
      {/* Inline keyframes */}
      <style>{`
        @keyframes fadeSlideIn {
          from { opacity: 0; transform: translateY(12px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        .animate-fadeSlideIn {
          animation: fadeSlideIn 0.3s ease-out both;
        }
        @keyframes fadeIn {
          from { opacity: 0; }
          to   { opacity: 1; }
        }
        .animate-fadeIn {
          animation: fadeIn 0.2s ease-out both;
        }
      `}</style>

      {/* Header */}
      <div className="shrink-0 px-6 py-4 border-b border-border">
        <h1 className="text-lg font-semibold text-foreground">Quick Start</h1>
        <p className="text-sm text-muted-foreground">Start a conversation with an agent in a few steps</p>
      </div>

      {/* Chat area */}
      <div className="flex-1 overflow-y-auto px-6 py-5 space-y-4">
        {bubbles.map((b) => (
          <Bubble key={b.id} role={b.role}>
            {b.content}
          </Bubble>
        ))}

        {/* Typing indicator */}
        {typing && <TypingDots />}

        {/* ── Interactive widgets ───────────────────────────────────── */}

        {/* Agent picker */}
        {phase === "pick-agent" && widgetVisible && (
          <div className="flex items-start gap-2.5 animate-fadeSlideIn">
            <BotAvatar />
            <div className="bg-muted rounded-2xl rounded-bl-md px-4 py-3 space-y-3 max-w-[75%]">
              {agents.length > 0 ? (
                <>
                  <select
                    value={selectedAgentId ?? ""}
                    onChange={(e) => setSelectedAgentId(e.target.value)}
                    className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                  >
                    {agents.map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name} ({truncateId(a.id)})
                      </option>
                    ))}
                  </select>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={handleConfirmAgent}
                      disabled={!selectedAgentId}
                      className="px-4 py-1.5 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                    >
                      Confirm
                    </button>
                    <span className="text-xs text-muted-foreground">or</span>
                    <button
                      onClick={handleOpenAgentEditor}
                      className="px-3 py-1.5 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors flex items-center gap-1.5"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                      </svg>
                      Create new
                    </button>
                  </div>
                </>
              ) : (
                <button
                  onClick={handleOpenAgentEditor}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Create agent
                </button>
              )}
            </div>
          </div>
        )}

        {/* Agent YAML editor */}
        {phase === "edit-agent" && widgetVisible && (
          <InlineEditor
            value={editorText}
            onChange={(v) => { setEditorText(v); setEditorError(null); }}
            format={editorFormat}
            onFormatChange={switchEditorFormat}
            error={editorError}
            submitting={editorSubmitting}
            onSubmit={handleSubmitAgent}
            onCancel={handleCancelAgentEditor}
            submitLabel="Create agent"
          />
        )}

        {/* Environment picker */}
        {phase === "pick-env" && widgetVisible && (
          <div className="flex items-start gap-2.5 animate-fadeSlideIn">
            <BotAvatar />
            <div className="bg-muted rounded-2xl rounded-bl-md px-4 py-3 space-y-3 max-w-[75%]">
              {environments.length > 0 ? (
                <>
                  <select
                    value={selectedEnvId ?? ""}
                    onChange={(e) => setSelectedEnvId(e.target.value)}
                    className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                  >
                    {environments.map((e) => (
                      <option key={e.id} value={e.id}>
                        {e.name} ({truncateId(e.id)})
                      </option>
                    ))}
                  </select>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={handleConfirmEnv}
                      disabled={!selectedEnvId}
                      className="px-4 py-1.5 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
                    >
                      Confirm
                    </button>
                    <span className="text-xs text-muted-foreground">or</span>
                    <button
                      onClick={handleOpenEnvForm}
                      className="px-3 py-1.5 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors flex items-center gap-1.5"
                    >
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                      </svg>
                      Create new
                    </button>
                  </div>
                </>
              ) : (
                <button
                  onClick={handleOpenEnvForm}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Create environment
                </button>
              )}
            </div>
          </div>
        )}

        {/* Environment simple form */}
        {phase === "edit-env" && widgetVisible && (
          <div className="flex items-start gap-2.5 animate-fadeSlideIn">
            <BotAvatar />
            <div className="bg-muted rounded-2xl rounded-bl-md px-4 py-4 space-y-3 w-[360px] max-w-[85%]">
              {/* Name */}
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Name</label>
                <input
                  type="text"
                  value={envFormName}
                  onChange={(e) => { setEnvFormName(e.target.value); setEnvFormError(null); }}
                  placeholder="e.g. My sandbox"
                  maxLength={50}
                  className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                />
              </div>

              {/* Networking */}
              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1">Networking</label>
                <select
                  value={envFormNetworking}
                  onChange={(e) => setEnvFormNetworking(e.target.value as NetworkingType)}
                  className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                >
                  <option value="limited">Limited</option>
                  <option value="unrestricted">Unrestricted</option>
                </select>
                <p className="text-xs text-muted-foreground mt-1">
                  {envFormNetworking === "limited"
                    ? "Only allowed hosts can be accessed."
                    : "Full internet access."}
                </p>
              </div>

              {/* Error */}
              {envFormError && (
                <div className="px-3 py-2 text-xs text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
                  {envFormError}
                </div>
              )}

              {/* Actions */}
              <div className="flex items-center justify-end gap-2 pt-1">
                <button
                  onClick={handleCancelEnvForm}
                  className="px-3 py-1.5 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSubmitEnv}
                  disabled={envFormSubmitting}
                  className="px-4 py-1.5 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                >
                  {envFormSubmitting && <Spinner />}
                  Create
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Session active — "View session detail" button */}
        {phase === "active" && sessionId && widgetVisible && (
          <div className="flex items-start gap-2.5 animate-fadeSlideIn">
            <BotAvatar />
            <div className="bg-muted rounded-2xl rounded-bl-md px-4 py-3">
              <button
                onClick={() => router.push(`/sessions/${sessionId}`)}
                className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
                View session detail
              </button>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Message input bar */}
      <div className="shrink-0 border-t border-border px-6 py-3">
        <div className={`flex items-end gap-3 transition-opacity ${!inputEnabled ? "opacity-50" : "opacity-100"}`}>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={
              inputEnabled
                ? "Type a message... (Enter to send, Shift+Enter for new line)"
                : "Complete the steps above to start chatting"
            }
            disabled={!inputEnabled || sending}
            rows={1}
            className="flex-1 px-4 py-2.5 text-sm bg-card border border-border rounded-xl placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors resize-none disabled:cursor-not-allowed"
            style={{ minHeight: "42px", maxHeight: "120px" }}
          />
          <button
            onClick={handleSend}
            disabled={!inputEnabled || !message.trim() || sending}
            className="px-4 py-2.5 text-sm font-medium text-white bg-primary rounded-xl hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 shrink-0"
          >
            {sending ? (
              <Spinner />
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            )}
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
