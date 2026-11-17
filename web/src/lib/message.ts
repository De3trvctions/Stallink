import { useCallback, useState } from "react";

export type AppMessage = { kind: "success" | "error"; text: string };

export function messageClass(message: AppMessage | null): string {
  if (!message) return "form-message";
  return message.kind === "error" ? "form-message form-message--error" : "form-message form-message--success";
}

// Explicit success/error message state, replacing the old Chinese-substring
// color guessing that misclassified messages containing words like "没有".
// Setters are stable so they can be listed in useEffect dependency arrays.
export function useMessage() {
  const [message, setMessageState] = useState<AppMessage | null>(null);
  const setMessage = useCallback((next: AppMessage | null) => setMessageState(next), []);
  const setSuccess = useCallback((text: string | null) => {
    setMessageState(text ? { kind: "success", text } : null);
  }, []);
  const setError = useCallback((text: string | null) => {
    setMessageState(text ? { kind: "error", text } : null);
  }, []);
  return { message, setMessage, setSuccess, setError };
}
