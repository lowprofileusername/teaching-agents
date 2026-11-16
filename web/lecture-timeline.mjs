export function intervalAt(entries,time){
 const active=entries.filter(x=>Number(x.t0)<=time&&Number(x.t1)>time).sort((a,b)=>(b.priority||0)-(a.priority||0)||b.t0-a.t0||a.t1-b.t1);
 const best=active[0];return{entry:best||null,ambiguous:!!best&&active.some(x=>(x.priority||0)===(best.priority||0)&&x.page_id!==best.page_id)};
}
export function imageAt(entry,images,time){if(!entry)return null;if(entry.asset_id)return{asset_id:entry.asset_id,kind:entry.image_kind||'分段代表截图'};const candidates=images.filter(x=>x.page_id===entry.page_id);const within=candidates.filter(x=>x.source_seconds>=entry.t0&&x.source_seconds<entry.t1);const usable=within.length?within:candidates;return usable.sort((a,b)=>Math.abs(a.source_seconds-time)-Math.abs(b.source_seconds-time))[0]||null}
export function audioLocalTime(chunk,time){return Math.max(0,Math.min(Number(chunk.encoded_duration)||chunk.t1-chunk.t0,time-chunk.t0))}
