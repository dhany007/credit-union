export type PageSize = 10 | 50 | 100 | "all";
export const pageSizes = [10, 50, 100, "all"] as const;
export function pageWindow<T>(rows: T[], size: PageSize, requestedPage: number) {
  const length = size === "all" ? Math.max(1, rows.length) : size;
  const pages = Math.max(1, Math.ceil(rows.length / length));
  const page = Math.max(0, Math.min(requestedPage, pages - 1));
  return { page, pages, rows: rows.slice(page * length, (page + 1) * length) };
}
export type CursorPage<T> = { items: T[]; next_cursor: string; start_date?: string; end_date?: string };
export type CursorCache<T> = { items: T[]; cursor: string; done: boolean; dates?: {start:string;end:string} };
export function emptyCursorCache<T>(): CursorCache<T> { return {items:[],cursor:"",done:false}; }
export function cursorURL(path:string, cache:CursorCache<unknown>, members=false) {
  const url=new URL(path,"http://local");
  if(cache.cursor) url.searchParams.set(members?"before_id":"cursor",cache.cursor);
  if(cache.dates){url.searchParams.set("period","custom");url.searchParams.set("start_date",cache.dates.start);url.searchParams.set("end_date",cache.dates.end);}
  return url.pathname+url.search;
}
export async function fillCursorCache<T>(cache:CursorCache<T>, target:number,
  fetchPage:(cache:CursorCache<T>)=>Promise<CursorPage<T>>, signal:AbortSignal,
  progress:(cache:CursorCache<T>)=>void) {
  let current=cache;
  while(!current.done && current.items.length<target) {
    if(signal.aborted) throw new DOMException("Aborted","AbortError");
    const page=await fetchPage(current);
    if(signal.aborted) throw new DOMException("Aborted","AbortError");
    if(page.next_cursor && page.next_cursor===current.cursor) throw Error("Invalid pagination cursor. Apply the filters again");
    current={items:[...current.items,...page.items],cursor:page.next_cursor,done:!page.next_cursor,
      dates:current.dates??(page.start_date&&page.end_date?{start:page.start_date,end:page.end_date}:undefined)};
    progress(current);
  }
  return current;
}
