export class ApiError extends Error {
  readonly status: number;
  constructor(message: string, status: number) { super(message); this.name = "ApiError"; this.status = status; }
}

export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`/api${path}`, {
    ...options,
    cache: "no-store",
    headers,
  });
  let data: unknown;
  try {
    data = await response.json();
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") throw error;
    throw new Error("The server could not be reached. Please try again.");
  }
  if (!response.ok) {
    const message =
      typeof data === "object" && data !== null && "error" in data &&
      typeof data.error === "string" && data.error
        ? data.error
        : "Operation failed. Please retry with the same reference.";
    throw new ApiError(message, response.status);
  }
  // Callers select the response type; endpoint-specific validation stays at their boundary.
  return data as T;
}
