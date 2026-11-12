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

export function selectLoanPreview(result: {key:string;preview?:Preview;error?:string}, key:string, enabled:boolean) {
  const preview = enabled && result.key === key && !result.error ? result.preview : undefined;
  const error = enabled && result.key === key ? result.error ?? "" : "";
  return { preview, displayPreview:result.preview, error, loading:enabled && !preview && !error };
}
