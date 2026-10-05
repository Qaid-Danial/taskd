import { handleMcp } from "./mcp.ts";

const SECRET = Deno.env.get("MCP_SECRET") ?? "";

// Compares two strings in constant time by comparing their SHA-256 hashes.
async function sameSecret(a: string, b: string): Promise<boolean> {
  const enc = new TextEncoder();
  const [x, y] = await Promise.all(
    [a, b].map((s) => crypto.subtle.digest("SHA-256", enc.encode(s))),
  );
  const ha = new Uint8Array(x);
  const hb = new Uint8Array(y);
  let diff = 0;
  for (let i = 0; i < ha.length; i++) diff |= ha[i] ^ hb[i];
  return diff === 0;
}

Deno.serve(async (req) => {
  // The URL looks like /mcp/<secret>; the last part is the secret.
  const given = new URL(req.url).pathname.split("/").filter(Boolean).at(-1) ??
    "";
  if (!SECRET || !(await sameSecret(given, SECRET))) {
    return new Response("Not found", { status: 404 });
  }
  return handleMcp(req);
});
