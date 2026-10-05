import { tools } from "./tools.ts";

const SUPPORTED_VERSIONS = ["2025-06-18", "2025-03-26"];

type RpcMessage = {
  jsonrpc?: string;
  id?: string | number | null;
  method?: string;
  params?: {
    protocolVersion?: string;
    name?: string;
    arguments?: Record<string, unknown>;
  };
};

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const errorText = (err: unknown) =>
  (err as { message?: string })?.message ?? String(err);

export async function handleMcp(req: Request): Promise<Response> {
  // This server only answers POSTed JSON-RPC messages.
  if (req.method !== "POST") {
    return new Response("Method not allowed", {
      status: 405,
      headers: { Allow: "POST" },
    });
  }

  let msg: RpcMessage;
  try {
    msg = await req.json();
  } catch {
    return json({
      jsonrpc: "2.0",
      id: null,
      error: { code: -32700, message: "Parse error" },
    }, 400);
  }

  // Messages without an id are notifications: no reply needed.
  if (msg.id === undefined) return new Response(null, { status: 202 });

  const reply = (result: unknown) =>
    json({ jsonrpc: "2.0", id: msg.id, result });
  const fail = (code: number, message: string) =>
    json({ jsonrpc: "2.0", id: msg.id, error: { code, message } });

  switch (msg.method) {
    case "initialize": {
      const asked = msg.params?.protocolVersion ?? "";
      return reply({
        protocolVersion: SUPPORTED_VERSIONS.includes(asked)
          ? asked
          : SUPPORTED_VERSIONS[0],
        capabilities: { tools: {} },
        serverInfo: { name: "todo-mcp", version: "0.1.0" },
      });
    }

    case "ping":
      return reply({});

    case "tools/list":
      return reply({
        tools: tools.map((t) => ({
          name: t.name,
          description: t.description,
          inputSchema: t.inputSchema,
        })),
      });

    case "tools/call": {
      const tool = tools.find((t) => t.name === msg.params?.name);
      if (!tool) return fail(-32602, `Unknown tool: ${msg.params?.name}`);
      try {
        const result = await tool.run(msg.params?.arguments ?? {});
        return reply({
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        });
      } catch (err) {
        return reply({
          content: [{ type: "text", text: `Error: ${errorText(err)}` }],
          isError: true,
        });
      }
    }

    default:
      return fail(-32601, `Method not found: ${msg.method}`);
  }
}
