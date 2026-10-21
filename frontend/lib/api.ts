export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`/api${path}`, {
    ...options,
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  const data = await response.json().catch(() => ({
    error: "The server could not be reached. Please try again.",
  }));
  if (!response.ok)
    throw new Error(
      data.error || "Operation failed. Please retry with the same reference.",
    );
  return data;
}
