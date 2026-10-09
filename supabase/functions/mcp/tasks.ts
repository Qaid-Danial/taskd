// All database work lives here. Every query is pinned to OWNER.
import { createClient } from "npm:@supabase/supabase-js@2";

const db = createClient(
  Deno.env.get("SUPABASE_URL")!,
  Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!,
);
const OWNER = Deno.env.get("OWNER_USER_ID")!;
const TZ = "Asia/Kuala_Lumpur";

type Args = Record<string, unknown>;

// "Today" in Malaysia time, as YYYY-MM-DD.
export function today(): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: TZ }).format(new Date());
}

const dateArg = (value: string) => (value === "today" ? today() : value);

// The only fields Claude may set. Anything else in the input is dropped.
const EDITABLE = [
  "title",
  "notes",
  "status",
  "priority",
  "area",
  "tags",
  "planned_for",
  "due_date",
  "estimate_minutes",
  "sort_order",
];

function pick(input: Args): Args {
  const out: Args = {};
  for (const key of EDITABLE) {
    if (input[key] === undefined) continue;
    const value = input[key];
    const isDate = key === "planned_for" || key === "due_date";
    out[key] = isDate && typeof value === "string" ? dateArg(value) : value;
  }
  return out;
}

export async function listTasks(args: Args) {
  const statuses = Array.isArray(args.status) && args.status.length > 0
    ? (args.status as string[])
    : ["todo", "doing"];

  let query = db.from("tasks").select("*")
    .eq("user_id", OWNER)
    .in("status", statuses);
  if (typeof args.area === "string") query = query.eq("area", args.area);
  if (typeof args.planned_for === "string") {
    query = query.eq("planned_for", dateArg(args.planned_for));
  }
  if (typeof args.due_before === "string") {
    query = query.lte("due_date", dateArg(args.due_before));
  }
  if (args.unscheduled === true) {
    query = query.is("planned_for", null).is("due_date", null);
  }
  const limit = Math.min(Number(args.limit ?? 100), 200);
  const { data, error } = await query.order("priority").order("sort_order")
    .limit(limit);
  if (error) throw error;
  return { today: today(), count: data.length, tasks: data };
}

export async function getSummary() {
  const t = today();
  const { data, error } = await db.from("tasks")
    .select("area, planned_for, due_date, estimate_minutes")
    .eq("user_id", OWNER)
    .in("status", ["todo", "doing"]);
  if (error) throw error;

  const byArea: Record<string, number> = {};
  let overdue = 0, plannedToday = 0, minutesToday = 0, unscheduled = 0;
  for (const task of data) {
    const area = task.area ?? "none";
    byArea[area] = (byArea[area] ?? 0) + 1;
    if (task.due_date && task.due_date < t) overdue++;
    if (task.planned_for === t) {
      plannedToday++;
      minutesToday += task.estimate_minutes ?? 0;
    }
    if (!task.planned_for && !task.due_date) unscheduled++;
  }
  return {
    today: t,
    open_tasks: data.length,
    by_area: byArea,
    overdue,
    planned_today: plannedToday,
    minutes_planned_today: minutesToday,
    unscheduled,
  };
}

export async function addTask(args: Args) {
  if (typeof args.title !== "string" || args.title.trim() === "") {
    throw new Error("title is required");
  }
  const { data, error } = await db.from("tasks")
    .insert({ ...pick(args), user_id: OWNER })
    .select()
    .single();
  if (error) throw error;
  return data;
}

export async function updateTask(args: Args) {
  if (typeof args.id !== "string") throw new Error("id is required");
  const fields = pick(args);
  if (Object.keys(fields).length === 0) {
    throw new Error("no editable fields given");
  }

  const { data, error } = await db.from("tasks")
    .update(fields)
    .eq("id", args.id)
    .eq("user_id", OWNER)
    .select()
    .single();
  if (error) throw error;
  return data;
}

export const completeTask = (args: Args) =>
  updateTask({ id: args.id, status: "done" });
export const archiveTask = (args: Args) =>
  updateTask({ id: args.id, status: "archived" });

export async function reorderTasks(args: Args) {
  const items = args.items;
  if (!Array.isArray(items) || items.length === 0 || items.length > 50) {
    throw new Error("items must be a list of 1 to 50 tasks");
  }
  const updated = [];
  for (const item of items as Args[]) {
    const change: Args = { id: item.id };
    if (item.sort_order !== undefined) change.sort_order = item.sort_order;
    if (item.priority !== undefined) change.priority = item.priority;
    updated.push(await updateTask(change));
  }
  return updated;
}

export async function getTaskHistory(args: Args) {
  if (typeof args.id !== "string") throw new Error("id is required");
  const limit = Math.min(Number(args.limit ?? 10), 50);
  const { data, error } = await db.from("task_events")
    .select("actor, action, old_row, new_row, created_at")
    .eq("task_id", args.id)
    .eq("user_id", OWNER)
    .order("created_at", { ascending: false })
    .limit(limit);
  if (error) throw error;
  return data;
}
