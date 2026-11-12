"use client";

import { useEffect, useRef, useState, type ReactNode, type CSSProperties } from "react";
import { Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { ErrorNotice } from "@/components/modal";
import { cursorURL, emptyCursorCache, fillCursorCache, pageSizes, pageWindow, type CursorCache, type CursorPage, type PageSize } from "@/lib/table-pagination";

export function TableViewport({children,label,loading=false,stale=false,statusText}:{children:ReactNode;label:string;loading?:boolean;stale?:boolean;statusText?:string}) {
  const {t}=usePreferences();const root=useRef<HTMLDivElement>(null);
  const [height,setHeight]=useState(560);const [captionHeight,setCaptionHeight]=useState(0);const measured=useRef<{width:number;height:number;caption:number}|null>(null);
  useEffect(()=>{
    const table=root.current?.querySelector("table");if(!table)return;
    const observer=new ResizeObserver(()=>{
      const rows=Array.from(table.querySelectorAll("tbody tr")).filter(row=>!row.querySelector("td[colspan]")).slice(0,10);
      const heights=rows.map(row=>row.getBoundingClientRect().height);
      const caption=table.querySelector("caption");
      const captionSize=caption&&!caption.classList.contains("sr-only")?caption.getBoundingClientRect().height:0;
      const width=Math.round(table.getBoundingClientRect().width);
      if(measured.current?.width===width) return;
      const head=(table.querySelector("thead")?.getBoundingClientRect().height??0)+captionSize;
      const rowHeight=heights[0]??48;
      const next=Math.ceil(head+heights.reduce((a,b)=>a+b,0)+Math.max(0,10-rows.length)*rowHeight);
      if(rows.length) measured.current={width,height:next,caption:captionSize};
      setCaptionHeight(captionSize);setHeight(previous=>previous===next?previous:next);
    });observer.observe(table);return()=>observer.disconnect();
  },[children]);
  return <div className="relative"><div ref={root} className="table-viewport" style={{height,"--table-caption-height":captionHeight+"px"} as CSSProperties} role="region" aria-label={label} aria-busy={loading} tabIndex={0}>
    <div inert={loading||stale||undefined} className={loading||stale?"opacity-40":undefined}>{children}</div>
  </div>{(loading||stale)&&<div className="absolute inset-0 z-20 grid place-content-center justify-items-center gap-2 bg-background/50 p-4 text-center text-sm" role="status">
    {loading&&<Loader2 aria-hidden="true" className="animate-spin" size={20}/>}<span>{statusText??t("Memuat...")}</span>
  </div>}</div>;
}
export function PageControls({size,onSize,page,onPage,hasNext,count,disabled=false}:{size:PageSize;onSize:(size:PageSize)=>void;page:number;onPage:(page:number)=>void;hasNext:boolean;count:number;disabled?:boolean}) {
  const {t}=usePreferences();
  return <div className="no-print flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
    <label className="flex items-center gap-2"><span>{t("Baris per halaman")}</span><select className="h-9 rounded-md border bg-card px-2" value={size} disabled={disabled} onChange={e=>onSize(e.target.value==="all"?"all":Number(e.target.value) as PageSize)}>
      {pageSizes.map(value=><option key={value} value={value}>{value==="all"?t("Semua"):value}</option>)}
    </select></label>
    <span role="status">{t("Halaman {page} · {count} data",{page:page+1,count})}</span>
    <div className="flex gap-2"><Button type="button" variant="outline" size="sm" disabled={disabled||page===0} onClick={()=>onPage(page-1)}>{t("Sebelumnya")}</Button><Button type="button" variant="outline" size="sm" disabled={disabled||!hasNext} onClick={()=>onPage(page+1)}>{t("Berikutnya")}</Button></div>
  </div>;
}
export function LocalRows<T>({rows,label,children,loading=false,stale=false,statusText,printAll=false,resetKey=""}:{rows:T[];label:string;children:(rows:T[])=>ReactNode;loading?:boolean;stale?:boolean;statusText?:string;printAll?:boolean;resetKey?:string}) {
  const [size,setSize]=useState<PageSize>(10);const [position,setPosition]=useState({scope:resetKey,index:0});const index=position.scope===resetKey?position.index:0;const setIndex=(value:number)=>setPosition({scope:resetKey,index:value});
  const page=pageWindow(rows,size,index);
  return <><div className={printAll?"no-print":undefined}><TableViewport label={label} loading={loading} stale={stale} statusText={statusText}>{children(page.rows)}</TableViewport>
    <PageControls size={size} onSize={value=>{setSize(value);setIndex(0);}} page={page.page} onPage={setIndex} hasNext={page.page+1<page.pages} count={page.rows.length} disabled={loading||stale}/></div>{printAll&&<div className="print-only">{children(rows)}</div>}</>;
}
export function ServerRows<T>({path,label,children,members=false}:{path:string;label:string;children:(rows:T[],dates?:{start:string;end:string})=>ReactNode;members?:boolean}) {
  const [size,setSize]=useState<PageSize>(10);const [position,setPosition]=useState({path,index:0});const index=position.path===path?position.index:0;const setIndex=(value:number)=>setPosition({path,index:value});const [attempt,setAttempt]=useState(0);
  const cache=useRef<{path:string;value:CursorCache<T>}>({path:"",value:emptyCursorCache<T>()});
  const [result,setResult]=useState<{path:string;value:CursorCache<T>;key:string;error:string}>({path:"",value:emptyCursorCache<T>(),key:"",error:""});
  const target=size==="all"?Infinity:(index+1)*size;const key=path+":"+attempt;
  useEffect(()=>{
    const controller=new AbortController();
    const initial=cache.current.path===path?cache.current.value:emptyCursorCache<T>();
    if(initial.done||initial.items.length>=target)return;
    fillCursorCache(initial,target,async current=>{
      const url=cursorURL(path,current,members);
      if(!members)return api<CursorPage<T>>(url,{signal:controller.signal});
      const rows=await api<(T&{id:number})[]>(url,{signal:controller.signal});
      return {items:rows,next_cursor:rows.length===100?String(rows[rows.length-1].id):""};
    },controller.signal,current=>{
      cache.current={path,value:current};setResult({path,value:current,key,error:""});
    }).catch(error=>{if(!controller.signal.aborted)setResult({path,value:cache.current.path===path?cache.current.value:initial,key,error:error instanceof Error?error.message:"Failed to fetch"});});
    return()=>controller.abort();
  },[path,target,members,key]);
  const current=result.path===path?result.value:emptyCursorCache<T>();
  const error=result.key===key?result.error:"";const ready=result.path===path&&(current.done||current.items.length>=target);
  const loading=!ready&&!error;
  const display=result.path===path?current.items:result.value.items;
  const page=pageWindow(display,size,index);
  const {t}=usePreferences();
  return <section className="space-y-2"><ErrorNotice error={error}/>
    <TableViewport label={label} loading={loading} stale={!!error} statusText={loading?t("Memuat data..."):error?t("Gagal memuat data. Silakan coba lagi."):undefined}>{children(page.rows,current.dates)}</TableViewport>
    {error&&<Button variant="outline" onClick={()=>setAttempt(value=>value+1)}>{t("Coba lagi")}</Button>}
    <PageControls size={size} onSize={value=>{setSize(value);setIndex(0);}} page={page.page} onPage={setIndex} hasNext={size!=="all"&&(display.length>(index+1)*size||!current.done)} count={page.rows.length} disabled={loading}/>
  </section>;
}
