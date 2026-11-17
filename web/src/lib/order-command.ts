import { browserJSON, type RequestResult } from "@/lib/api";

type StoredOrderCommand = {
  key: string;
  body: string;
  createdAt: number;
};

const prefix = "stallink.order-command.v1.";

function readCommand(scope: string): StoredOrderCommand | null {
  try {
    const value = localStorage.getItem(prefix + scope);
    if (!value) return null;
    const parsed = JSON.parse(value) as StoredOrderCommand;
    return typeof parsed.key === "string" && typeof parsed.body === "string" ? parsed : null;
  } catch {
    return null;
  }
}

export function pendingOrderCommand<T>(scope: string): { key: string; body: T } | null {
  const saved = readCommand(scope);
  if (!saved) return null;
  try {
    return { key: saved.key, body: JSON.parse(saved.body) as T };
  } catch {
    return null;
  }
}

export async function sendOrderCommand(
  scope: string,
  path: string,
  body: unknown,
): Promise<RequestResult> {
  const encodedBody = JSON.stringify(body);
  let command = readCommand(scope);
  if (command && command.body !== encodedBody) {
    return {
      ok: false,
      status: 409,
      data: { error: "command_payload_unresolved" } as Record<string, unknown>,
      category: "client",
      errorCode: "command_payload_unresolved",
    };
  }
  if (!command) {
    command = { key: crypto.randomUUID(), body: encodedBody, createdAt: Date.now() };
    try {
      localStorage.setItem(prefix + scope, JSON.stringify(command));
    } catch {
      return {
        ok: false,
        status: 500,
        data: { error: "command_storage_unavailable" } as Record<string, unknown>,
        category: "server",
        errorCode: "command_storage_unavailable",
      };
    }
  }

  // Keep the exact key and payload after network failures or 5xx responses.
  // The next attempt must replay this command instead of creating another one.
  const result = await browserJSON(path, "POST", JSON.parse(command.body), {
    "Idempotency-Key": command.key,
  });
  const errorCode = String(result.data.error ?? "");
  const terminalRejection = result.status >= 400 && result.status < 500 && !["command_in_progress", "idempotency_conflict"].includes(errorCode);
  if (result.ok || terminalRejection) {
    try {
      if (readCommand(scope)?.key === command.key) localStorage.removeItem(prefix + scope);
    } catch {
      // A successful server response remains authoritative if storage is lost.
    }
  }
  return result;
}

export function clearPendingOrderCommand(scope: string) {
  try {
    localStorage.removeItem(prefix + scope);
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}

export function newClientOrderId() {
  return crypto.randomUUID();
}
