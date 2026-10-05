import * as tasks from "./tasks.ts";

type Tool = {
  name: string;
  description: string;
  inputSchema: Record<string, unknown>;
  run: (args: Record<string, unknown>) => Promise<unknown>;
};

const PRIORITY =
  "Priority: 1 = urgent and important, 2 = important, 3 = normal, 4 = someday.";
const DATES =
  "Dates are YYYY-MM-DD or the word 'today', in Malaysia time (Asia/Kuala_Lumpur).";

const taskFields = {
  title: { type: "string", maxLength: 200 },
  notes: { type: "string" },
  priority: { type: "integer", minimum: 1, maximum: 4 },
  area: {
    type: "string",
    description: "Life area, e.g. fyp, internship, job-search, freelance",
  },
  tags: { type: "array", items: { type: "string" } },
  planned_for: {
    type: "string",
    description: "Day the user intends to work on it",
  },
  due_date: { type: "string", description: "Hard deadline" },
  estimate_minutes: { type: "integer", minimum: 1 },
};

const idOnly = {
  type: "object",
  properties: { id: { type: "string", description: "Task id (uuid)" } },
  required: ["id"],
};

export const tools: Tool[] = [
  {
    name: "list_tasks",
    description:
      `List the user's tasks. Defaults to open tasks (todo and doing). ${DATES} ${PRIORITY} The result includes today's date.`,
    inputSchema: {
      type: "object",
      properties: {
        status: {
          type: "array",
          items: {
            type: "string",
            enum: ["todo", "doing", "done", "archived"],
          },
        },
        area: { type: "string" },
        planned_for: { type: "string" },
        due_before: {
          type: "string",
          description: "Tasks due on or before this date",
        },
        limit: { type: "integer", maximum: 200 },
      },
    },
    run: tasks.listTasks,
  },
  {
    name: "get_summary",
    description:
      "Overview of open tasks: counts by area, overdue count, tasks and minutes planned for today. Use this first for reviews.",
    inputSchema: { type: "object", properties: {} },
    run: () => tasks.getSummary(),
  },
  {
    name: "add_task",
    description: `Create a task. ${DATES} ${PRIORITY}`,
    inputSchema: {
      type: "object",
      properties: taskFields,
      required: ["title"],
    },
    run: tasks.addTask,
  },
  {
    name: "update_task",
    description:
      `Change fields on one task. Only include the fields to change. ${DATES} ${PRIORITY}`,
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        status: { type: "string", enum: ["todo", "doing", "done", "archived"] },
        sort_order: { type: "integer" },
        ...taskFields,
      },
      required: ["id"],
    },
    run: tasks.updateTask,
  },
  {
    name: "complete_task",
    description: "Mark a task as done.",
    inputSchema: idOnly,
    run: tasks.completeTask,
  },
  {
    name: "archive_task",
    description:
      "Archive a task the user no longer needs. Tasks are never deleted.",
    inputSchema: idOnly,
    run: tasks.archiveTask,
  },
  {
    name: "reorder_tasks",
    description:
      "Set sort_order and/or priority on up to 50 tasks at once. Lower sort_order shows first.",
    inputSchema: {
      type: "object",
      properties: {
        items: {
          type: "array",
          maxItems: 50,
          items: {
            type: "object",
            properties: {
              id: { type: "string" },
              sort_order: { type: "integer" },
              priority: { type: "integer", minimum: 1, maximum: 4 },
            },
            required: ["id"],
          },
        },
      },
      required: ["items"],
    },
    run: tasks.reorderTasks,
  },
  {
    name: "get_task_history",
    description:
      "Recent changes to one task, newest first, including who made them (user or claude). Use it to undo a change.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string" },
        limit: { type: "integer", maximum: 50 },
      },
      required: ["id"],
    },
    run: tasks.getTaskHistory,
  },
];
