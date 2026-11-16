export function intervalAt(entries,time){
 const active=entries.filter(x=>Number(x.t0)<=time&&Number(x.t1)>time).sort((a,b)=>(b.priority||0)-(a.priority||0)||b.t0-a.t0||a.t1-b.t1);
 const best=active[0];return{entry:best||null,ambiguous:!!best&&active.some(x=>(x.priority||0)===(best.priority||0)&&x.page_id!==best.page_id)};
}
export function imageAt(entry,images,time){if(!entry)return null;if(entry.asset_id)return{asset_id:entry.asset_id,kind:entry.image_kind||'分段代表截图'};const candidates=images.filter(x=>x.page_id===entry.page_id);const within=candidates.filter(x=>x.source_seconds>=entry.t0&&x.source_seconds<entry.t1);const usable=within.length?within:candidates;return usable.sort((a,b)=>Math.abs(a.source_seconds-time)-Math.abs(b.source_seconds-time))[0]||null}
export function audioLocalTime(chunk,time){return Math.max(0,Math.min(Number(chunk.encoded_duration)||chunk.t1-chunk.t0,time-chunk.t0))}

export function transcriptVersions(rows){
 const groups=new Map();for(const u of rows){if(String(u.quality_status).includes('superseded'))continue;const v=u.version||'';if(!groups.has(v))groups.set(v,[]);groups.get(v).push(u)}
 const covered=rs=>{let end=0,total=0;for(const u of [...rs].sort((a,b)=>a.t0-b.t0)){total+=Math.max(0,u.t1-Math.max(end,u.t0));end=Math.max(end,u.t1)}return total};
 const rank=v=>v==='comprehensive_editorial_v1'?100000000:covered(groups.get(v));
 return [...groups.keys()].sort((a,b)=>rank(b)-rank(a));
}
export function transcriptLabel(v){return v==='comprehensive_editorial_v1'?'综合整理字幕（含疑词，未听校）':v||'未标注版本'}

export function audioQualityLabel(status){
 if(status==='encoded_decode_checked_not_listened')return '音频编码及解码检查通过，未做原声听辨';
 if(!status)return '音频未标注听辨状态';
 if(/not_listened|unverified|未听/.test(status))return '音频未完成原声听辨';
 return '音频核验状态：'+status;
}

export function intervalSelection(entry,duration){
 if(!entry||entry.t0===null||entry.t1===null)return null;
 const a=Number(entry.t0),b=Number(entry.t1),end=Number(duration);
 return Number.isFinite(a)&&Number.isFinite(b)&&Number.isFinite(end)&&a>=0&&b>a&&a<end?[a,Math.min(b,end)]:null;
}
