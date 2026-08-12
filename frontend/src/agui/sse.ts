// AG-UI 协议的 SSE 客户端：把后端推送的 text/event-stream 解析成事件对象。

export interface AguiEvent {
  type: string;
  [key: string]: unknown;
}

function parseSseFrame(frame: string): AguiEvent | null {
  const dataLines: string[] = [];
  for (const line of frame.split(/\r?\n/)) {
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trimStart());
    }
  }
  const data = dataLines.join("\n").trim();
  if (!data) {
    return null;
  }
  try {
    return JSON.parse(data) as AguiEvent;
  } catch {
    return { type: "RAW", raw: data };
  }
}

export async function streamAguiSse(
  url: string,
  payload: unknown,
  opts: {
    signal?: AbortSignal;
    onEvent: (evt: AguiEvent) => void;
  },
): Promise<void> {
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      accept: "text/event-stream",
    },
    body: JSON.stringify(payload),
    signal: opts.signal,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`请求失败 (${res.status}): ${text || res.statusText}`);
  }
  if (!res.body) {
    throw new Error("响应体为空");
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  for (;;) {
    const { value, done } = await reader.read();
    if (done) {
      break;
    }
    buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, "\n");

    let idx: number;
    while ((idx = buffer.indexOf("\n\n")) !== -1) {
      const frame = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 2);
      const evt = parseSseFrame(frame);
      if (evt) {
        opts.onEvent(evt);
      }
    }
  }

  const tail = buffer.trim();
  if (tail) {
    const evt = parseSseFrame(tail);
    if (evt) {
      opts.onEvent(evt);
    }
  }
}
