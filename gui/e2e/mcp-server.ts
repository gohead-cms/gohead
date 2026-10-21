import { createServer, type IncomingMessage, type ServerResponse } from "node:http";
import type { AddressInfo } from "node:net";

// A minimal MCP server (Streamable HTTP, JSON responses) for the connected
// apps tests: two tools, a bearer token.

export const MCP_TOKEN = "e2e-mcp-token";

const TOOLS = [
  {
    name: "search_issues",
    title: "Search issues",
    description: "Search the tracker's issues",
    inputSchema: { type: "object", properties: { query: { type: "string" } }, required: ["query"] },
    annotations: { readOnlyHint: true },
  },
  {
    name: "create_issue",
    title: "Create an issue",
    description: "Open a new issue in the tracker",
    inputSchema: { type: "object", properties: { title: { type: "string" } }, required: ["title"] },
  },
];

type Message = { jsonrpc: "2.0"; id?: number | string; method?: string; params?: Record<string, unknown> };

function answer(msg: Message): unknown {
  switch (msg.method) {
    case "initialize":
      return {
        protocolVersion: (msg.params?.protocolVersion as string) ?? "2025-06-18",
        capabilities: { tools: {} },
        serverInfo: { name: "e2e-tracker", version: "1.0.0" },
      };
    case "tools/list":
      return { tools: TOOLS };
    case "tools/call":
      return { content: [{ type: "text", text: `called ${String(msg.params?.name)}` }] };
    case "ping":
      return {};
  }
  return undefined;
}

function read(req: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    let body = "";
    req.on("data", (c: Buffer) => (body += c.toString()));
    req.on("end", () => resolve(body));
    req.on("error", reject);
  });
}

async function handle(req: IncomingMessage, res: ServerResponse) {
  if (req.headers.authorization !== `Bearer ${MCP_TOKEN}`) {
    res.writeHead(401).end("unauthorized");
    return;
  }
  if (req.method === "DELETE") {
    res.writeHead(200).end();
    return;
  }
  if (req.method !== "POST") {
    res.writeHead(405).end();
    return;
  }
  const msg = JSON.parse(await read(req)) as Message;
  if (msg.id === undefined) {
    res.writeHead(202).end(); // a notification
    return;
  }
  const result = answer(msg);
  const reply =
    result === undefined
      ? { jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: `unknown method ${String(msg.method)}` } }
      : { jsonrpc: "2.0", id: msg.id, result };
  res.writeHead(200, { "Content-Type": "application/json", "Mcp-Session-Id": "e2e-session" }).end(JSON.stringify(reply));
}

/** Starts the server on a free port of 127.0.0.1. */
export async function startMcpServer(): Promise<{ url: string; close: () => Promise<void> }> {
  const server = createServer((req, res) => {
    handle(req, res).catch(() => res.writeHead(500).end());
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as AddressInfo;
  return {
    url: `http://127.0.0.1:${port}/mcp`,
    close: () => new Promise((resolve) => server.close(() => resolve())),
  };
}
