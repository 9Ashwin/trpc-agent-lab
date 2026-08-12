import { useRef, useState, type KeyboardEvent } from "react";
import ReactMarkdown from "react-markdown";
import { streamAguiSse, type AguiEvent } from "./agui/sse";
import "./App.css";

interface ToolCall {
  id: string;
  name: string;
  args: string;
  result?: string;
  status: "running" | "done";
}

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  thinking?: string;
  toolCalls: ToolCall[];
  status: "streaming" | "done" | "error";
}

function newId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function newThreadId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return newId("thread");
}

function str(v: unknown): string {
  return typeof v === "string" ? v : "";
}

export default function App() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);

  const threadIdRef = useRef(newThreadId());
  const abortRef = useRef<AbortController | null>(null);
  // AG-UI messageId/toolCallId -> 本地消息 id 的映射
  const msgMapRef = useRef<Record<string, string>>({});
  const toolMapRef = useRef<Record<string, { messageId: string; callId: string }>>({});
  // 当前这一轮正在流式输出的 assistant 消息
  const currentRef = useRef<{ id: string } | null>(null);

  const updateMessage = (id: string, updater: (m: Message) => Message) => {
    setMessages((prev) => prev.map((m) => (m.id === id ? updater(m) : m)));
  };

  // 返回当前 assistant 消息 id，不存在则新建一个占位消息。
  const ensureCurrentAssistant = (): string => {
    if (currentRef.current) {
      return currentRef.current.id;
    }
    const localId = newId("assistant");
    currentRef.current = { id: localId };
    setMessages((prev) => [
      ...prev,
      { id: localId, role: "assistant", content: "", toolCalls: [], status: "streaming" },
    ]);
    return localId;
  };

  const handleEvent = (evt: AguiEvent) => {
    switch (str(evt.type)) {
      case "TEXT_MESSAGE_START": {
        const messageId = str(evt.messageId) || newId("assistant");
        const localId = ensureCurrentAssistant();
        msgMapRef.current[messageId] = localId;
        break;
      }
      case "TEXT_MESSAGE_CONTENT": {
        const messageId = str(evt.messageId);
        const delta = str(evt.delta);
        const localId = msgMapRef.current[messageId] ?? currentRef.current?.id;
        if (!localId || !delta) {
          break;
        }
        updateMessage(localId, (m) => ({ ...m, content: m.content + delta }));
        break;
      }
      case "REASONING_MESSAGE_CONTENT":
      case "REASONING_MESSAGE_CHUNK": {
        const delta = str(evt.delta);
        const localId = currentRef.current?.id;
        if (!localId || !delta) {
          break;
        }
        updateMessage(localId, (m) => ({ ...m, thinking: (m.thinking ?? "") + delta }));
        break;
      }
      case "TOOL_CALL_START": {
        const toolCallId = str(evt.toolCallId) || newId("tool");
        const name = str(evt.toolCallName) || "tool";
        const localId = ensureCurrentAssistant();
        toolMapRef.current[toolCallId] = { messageId: localId, callId: toolCallId };
        updateMessage(localId, (m) => ({
          ...m,
          toolCalls: [...m.toolCalls, { id: toolCallId, name, args: "", status: "running" }],
        }));
        break;
      }
      case "TOOL_CALL_ARGS": {
        const toolCallId = str(evt.toolCallId);
        const delta = str(evt.delta);
        const rec = toolMapRef.current[toolCallId];
        if (!rec || !delta) {
          break;
        }
        updateMessage(rec.messageId, (m) => ({
          ...m,
          toolCalls: m.toolCalls.map((t) =>
            t.id === rec.callId ? { ...t, args: t.args + delta } : t,
          ),
        }));
        break;
      }
      case "TOOL_CALL_RESULT": {
        const toolCallId = str(evt.toolCallId);
        const result = str(evt.content);
        const rec = toolMapRef.current[toolCallId];
        if (!rec) {
          break;
        }
        updateMessage(rec.messageId, (m) => ({
          ...m,
          toolCalls: m.toolCalls.map((t) =>
            t.id === rec.callId ? { ...t, result, status: "done" } : t,
          ),
        }));
        break;
      }
      case "RUN_ERROR": {
        const errMsg = str(evt.message) || "运行出错";
        const localId = currentRef.current?.id;
        if (localId) {
          updateMessage(localId, (m) => ({
            ...m,
            content: m.content ? `${m.content}\n\n⚠️ ${errMsg}` : `⚠️ ${errMsg}`,
            status: "error",
          }));
        } else {
          setMessages((prev) => [
            ...prev,
            { id: newId("error"), role: "assistant", content: `⚠️ ${errMsg}`, toolCalls: [], status: "error" },
          ]);
        }
        break;
      }
      default:
        // RUN_STARTED / RUN_FINISHED / TEXT_MESSAGE_END / STATE_* 等无需处理
        break;
    }
  };

  const settle = () => {
    setMessages((prev) =>
      prev.map((m) => (m.status === "streaming" ? { ...m, status: "done" } : m)),
    );
  };

  const send = async () => {
    const text = input.trim();
    if (!text || busy) {
      return;
    }
    setInput("");
    setBusy(true);

    const userMsg: Message = { id: newId("user"), role: "user", content: text, toolCalls: [], status: "done" };
    setMessages((prev) => [...prev, userMsg]);

    msgMapRef.current = {};
    toolMapRef.current = {};
    currentRef.current = null;

    const abort = new AbortController();
    abortRef.current = abort;

    const payload = {
      threadId: threadIdRef.current,
      runId: newId("run"),
      messages: [{ role: "user", content: text }],
    };

    try {
      await streamAguiSse("/agui", payload, {
        signal: abort.signal,
        onEvent: handleEvent,
      });
      settle();
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        setMessages((prev) => [
          ...prev,
          { id: newId("error"), role: "assistant", content: `⚠️ ${(err as Error).message}`, toolCalls: [], status: "error" },
        ]);
      }
      settle();
    } finally {
      setBusy(false);
      abortRef.current = null;
    }
  };

  const stop = () => {
    abortRef.current?.abort();
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void send();
    }
  };

  return (
    <div className="app">
      <header className="app-header">
        <div className="brand">
          <span className="brand-dot" />
          <h1>Pro-Me</h1>
        </div>
        <span className="app-subtitle">DeepSeek Agent · trpc-agent-go</span>
      </header>

      <main className="chat">
        {messages.length === 0 && (
          <div className="empty">
            <p className="empty-title">你好，我是 DeepSeek Agent 👋</p>
            <p>我会调用工具来回答你。试试：</p>
            <p className="empty-hint">
              <em>计算 (12 + 34) × 5</em>
              <em>纽约现在是几点？</em>
            </p>
          </div>
        )}

        {messages.map((m) =>
          m.role === "user" ? (
            <div key={m.id} className="msg user">
              <div className="bubble">{m.content}</div>
            </div>
          ) : (
            <div key={m.id} className="msg assistant">
              <div className="bubble">
                {m.thinking && (
                  <details className="thinking" open={m.status === "streaming"}>
                    <summary>思考过程</summary>
                    <p>{m.thinking}</p>
                  </details>
                )}
                {m.toolCalls.length > 0 && (
                  <div className="toolcalls">
                    {m.toolCalls.map((t) => (
                      <div key={t.id} className={`tool ${t.status}`}>
                        <div className="tool-head">
                          <span className="tool-icon">🔧</span>
                          <span className="tool-name">{t.name}</span>
                          {t.status === "running" && <span className="tool-spinner" />}
                        </div>
                        {t.args && <pre className="tool-args">{t.args}</pre>}
                        {t.result && <pre className="tool-result">{t.result}</pre>}
                      </div>
                    ))}
                  </div>
                )}
                {m.content ? (
                  <ReactMarkdown>{m.content}</ReactMarkdown>
                ) : (
                  m.status === "streaming" && <span className="cursor" />
                )}
              </div>
            </div>
          ),
        )}
      </main>

      <footer className="composer">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder="输入消息，Enter 发送，Shift+Enter 换行"
          rows={2}
        />
        {busy ? (
          <button className="btn stop" onClick={stop}>
            停止
          </button>
        ) : (
          <button className="btn send" onClick={() => void send()} disabled={!input.trim()}>
            发送
          </button>
        )}
      </footer>
    </div>
  );
}
