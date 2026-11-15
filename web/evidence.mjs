// Bound excerpts without dropping entire speakers or copying large diagnostic metadata.
const excerpt=(v,n)=>{const s=typeof v==='string'?v:'';return s.length>n?s.slice(0,n)+'〔摘录〕':s};
export function buildEvidence(r){return {
 scope:'当前所选资料库的本轮检索摘录；P/R均为各库编号，不是已核实的原PPT页码；未提供全文不等于原库无资料。',
 pages:(r.pages||[]).map(p=>({id:p.id,teacher:p.teacher,institution:p.institution,course:p.course,title:p.title,original_page_no:p.original_page_no,source_format:p.source_format,research_analysis_unverified:excerpt(p.research_analysis_unverified,900),page_facts:excerpt(p.page_text,2000),occurrences:(p.occurrences||[]).slice(0,3),transcript_sample_unverified:(p.utterances||[]).slice(0,2).map(u=>({text:excerpt(u.text,500),t0:u.t0,t1:u.t1,version:u.version,quality_status:u.quality_status,teacher_attribution:u.teacher_attribution})),official_pages:(p.official_pages||[]).slice(0,1).map(x=>({id:x.id,title:x.title,pdf_page_no:x.pdf_page_no,text:excerpt(x.direct_text||x.ocr_text,700),extraction:x.direct_text?'official_pdf_text':'official_pdf_ocr_unverified'})),issues:(p.issues||[]).slice(0,2).map(x=>({issue:excerpt(x.issue,300),status:x.status})),annotations:(p.annotations||[]).slice(0,2).map(x=>({original:excerpt(x.original,100),candidate:excerpt(x.candidate,100),status:x.status}))})),
 document_evidence:(r.document_hits||[]).map(h=>({id:h.id,document_id:h.document_id,title:h.title,pdf_page_no:h.pdf_page_no,text:excerpt(h.text,1800),authority_status:h.authority_status,edition:h.edition,extraction:h.extraction})),
 official_pdf_hits:(r.official_hits||[]).slice(0,4).map(h=>({id:h.id,title:h.title,teacher:h.teacher,pdf_page_no:h.pdf_page_no,direct_text:excerpt(h.direct_text,1000),ocr_text:excerpt(h.ocr_text,1000)})),
 raw_ocr_unverified:(r.raw_ocr||[]).slice(0,3).map(h=>({tag:h.tag,text_id:h.text_id,seconds:h.seconds,text:excerpt(h.text,1000)})),
 research_judgments:(r.research_hits||[]).slice(0,2).map(h=>({id:h.id,title:h.title,payload_json:excerpt(h.payload_json,1000)}))
};}
