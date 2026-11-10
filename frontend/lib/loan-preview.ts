import type { Preview } from "./loans";

// Ignore obsolete results even if a transport finishes after cancellation.
export function scheduleLoanPreview(
  request: (signal: AbortSignal) => Promise<Preview>,
  success: (preview: Preview) => void,
  failure: (error: unknown) => void,
) {
  const controller = new AbortController();
  const timer = setTimeout(() => {
    request(controller.signal).then(
      preview => { if (!controller.signal.aborted) success(preview); },
      error => { if (!controller.signal.aborted) failure(error); },
    );
  }, 350);
  return () => { clearTimeout(timer); controller.abort(); };
}
