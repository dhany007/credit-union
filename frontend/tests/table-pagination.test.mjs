import assert from "node:assert/strict";
import {test} from "node:test";
import {pageWindow,pageSizes,emptyCursorCache,fillCursorCache,cursorURL} from "../lib/table-pagination.ts";
import {selectLoanPreview} from "../lib/loan-preview.ts";
test("10/50/100/All pages preserve every record and clamp short last pages",()=>{
 const rows=Array.from({length:136},(_,i)=>i+1);
 assert.deepEqual(pageSizes,[10,50,100,"all"]);
 assert.deepEqual(pageWindow(rows,10,0).rows,rows.slice(0,10));
 assert.equal(pageWindow(rows,10,99).page,13);assert.equal(pageWindow(rows,10,99).rows.length,6);
 for(const size of [50,100]){const pages=pageWindow(rows,size,0).pages;assert.deepEqual(Array.from({length:pages},(_,i)=>pageWindow(rows,size,i).rows).flat(),rows);}
 assert.deepEqual(pageWindow(rows,"all",9),{page:0,pages:1,rows});
 assert.deepEqual(pageWindow([],10,9),{page:0,pages:1,rows:[]});
});
test("bounded cursor pages are buffered, not truncated by the display size",async()=>{
 const source=Array.from({length:65},(_,i)=>i+1);let requests=0;let cache=emptyCursorCache();
 const signal=new AbortController().signal;
 const fetch=async c=>{requests++;const start=Number(c.cursor||0),end=Math.min(start+20,65);return {items:source.slice(start,end),next_cursor:end<65?String(end):""};};
 cache=await fillCursorCache(cache,10,fetch,signal,()=>{});assert.equal(cache.items.length,20);
 cache=await fillCursorCache(cache,20,fetch,signal,()=>{});assert.equal(requests,1);
 cache=await fillCursorCache(cache,50,fetch,signal,()=>{});assert.equal(cache.items.length,60);
 cache=await fillCursorCache(cache,Infinity,fetch,signal,()=>{});assert.deepEqual(cache.items,source);assert.equal(cache.done,true);assert.equal(requests,4);
});
test("transaction dates and filters are frozen across cursor requests",()=>{
 const cache={...emptyCursorCache(),cursor:"opaque",dates:{start:"2026-11-01",end:"2026-11-11"}};
 const url=new URL(cursorURL("/transactions/search?period=today&account_number=1000000015",cache),"http://local");
 assert.equal(url.searchParams.get("period"),"custom");assert.equal(url.searchParams.get("start_date"),"2026-11-01");
 assert.equal(url.searchParams.get("end_date"),"2026-11-11");assert.equal(url.searchParams.get("account_number"),"1000000015");assert.equal(url.searchParams.get("cursor"),"opaque");
 const members=new URL(cursorURL("/members?search=Olivia",{...emptyCursorCache(),cursor:"42"},true),"http://local");
 assert.equal(members.searchParams.get("search"),"Olivia");assert.equal(members.searchParams.get("before_id"),"42");
});
test("cancelled responses cannot populate a different table scope",async()=>{
 const controller=new AbortController();let resolve;const snapshots=[];
 const promise=fillCursorCache(emptyCursorCache(),10,()=>new Promise(r=>{resolve=r;}),controller.signal,v=>snapshots.push(v));
 controller.abort();resolve({items:[1],next_cursor:""});
 await assert.rejects(promise,{name:"AbortError"});assert.deepEqual(snapshots,[]);
});
test("a repeated server cursor fails rather than looping forever",async()=>{
 await assert.rejects(fillCursorCache({items:[],cursor:"same",done:false},Infinity,async()=>({items:[1],next_cursor:"same"}),new AbortController().signal,()=>{}));
});
test("last schedule stays visible during loading but cannot authorize approval",()=>{
 const old={amount:1000000};
 const loading=selectLoanPreview({key:"old",preview:old},"new",true);
 assert.equal(loading.preview,undefined);assert.equal(loading.displayPreview,old);assert.equal(loading.loading,true);
 const failed=selectLoanPreview({key:"new",preview:old,error:"failed"},"new",true);
 assert.equal(failed.preview,undefined);assert.equal(failed.displayPreview,old);assert.equal(failed.loading,false);
 const fresh=selectLoanPreview({key:"new",preview:{amount:2000000}},"new",true);
 assert.equal(fresh.preview.amount,2000000);
 const invalid=selectLoanPreview({key:"new",preview:old},"new",false);
 assert.equal(invalid.preview,undefined);assert.equal(invalid.displayPreview,old);
});
