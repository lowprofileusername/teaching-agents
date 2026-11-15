#!/usr/bin/env python3
"""Portable read-only SQLite interface; Python 3 standard library only."""
import argparse,sqlite3,json,time,zlib,hashlib
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--db',required=True);s=p.add_subparsers(dest='action',required=True)
s.add_parser('schema');q=s.add_parser('search');q.add_argument('--keyword',action='append',required=True);q.add_argument('--tag',default='');q=s.add_parser('page');q.add_argument('id');q=s.add_parser('export');q.add_argument('--kind',choices=['pdf','asset'],required=True);q.add_argument('--id',required=True);q.add_argument('--out',required=True)
a=p.parse_args();dbpath=Path(a.db).resolve();c=sqlite3.connect(dbpath.as_uri()+'?mode=ro',uri=True,timeout=3);c.row_factory=sqlite3.Row;c.execute('pragma query_only=on');c.execute('pragma temp_store=memory')
deadline=time.monotonic()+5;c.set_progress_handler(lambda:time.monotonic()>deadline,10000)
def rows(sql,args=()):return [dict(x) for x in c.execute(sql,args)]
if a.action=='schema':out=rows("SELECT name,sql FROM sqlite_master WHERE type IN ('table','view') AND name NOT LIKE '%_data' AND name NOT LIKE '%_idx'")
elif a.action=='search':
 terms=[x.strip() for x in a.keyword if x.strip()][:8]
 if not terms:raise SystemExit('Need keywords')
 pred=' OR '.join(['instr(p.title||p.page_text,?)>0']*len(terms));out={'pages':rows('SELECT p.id,p.tag,p.page_no,p.original_page_no,p.title,p.page_text,v.teacher,v.course FROM page p JOIN video v ON v.tag=p.tag WHERE (?="" OR p.tag=?) AND ('+pred+') LIMIT 12',[a.tag,a.tag]+terms)}
 pred=' OR '.join(['instr(d.title||p.search_text,?)>0']*len(terms))
 try:out['reports']=rows('SELECT d.id AS document_id,d.title,d.authority_status,p.pdf_page_no,p.text FROM knowledge_document_page p JOIN knowledge_document d ON d.id=p.document_id WHERE (?="" OR d.video_tag=? OR d.video_tag IS NULL) AND ('+pred+') LIMIT 12',[a.tag,a.tag]+terms)
 except sqlite3.OperationalError as e:out['report_error']=str(e)
elif a.action=='page':
 out={'page':rows('SELECT * FROM page WHERE id=?',(a.id,)),'occurrences':rows('SELECT t0,t1 FROM page_occurrence WHERE page_id=? ORDER BY t0',(a.id,)),'utterances':rows('SELECT DISTINCT u.id,u.t0,u.t1,u.text,u.version,u.quality_status FROM page_utterance pu JOIN utterance u ON u.id=pu.utterance_id WHERE pu.page_id=? ORDER BY u.t0 LIMIT 500',(a.id,))}
else:
 c.set_progress_handler(None,0);table,key=('knowledge_document_pdf','document_id') if a.kind=='pdf' else ('asset','id');r=c.execute('SELECT content FROM '+table+' WHERE '+key+'=?',(a.id,)).fetchone()
 if not r:raise SystemExit('Media not found')
 b=r[0]
 try:codec=c.execute('SELECT codec,raw_sha256,raw_bytes FROM media_encoding WHERE table_name=? AND record_id=?',(table,a.id)).fetchone()
 except sqlite3.OperationalError:codec=None
 if codec:
  if codec['codec']!='zlib':raise SystemExit('Unsupported codec')
  b=zlib.decompress(b)
  if len(b)!=codec['raw_bytes'] or hashlib.sha256(b).hexdigest()!=codec['raw_sha256']:raise SystemExit('Media verification failed')
 dest=Path(a.out)
 if dest.exists():raise SystemExit('Output exists; choose a new filename')
 dest.write_bytes(b);out={'file':str(dest.resolve()),'bytes':len(b),'sha256':hashlib.sha256(b).hexdigest()}
print(json.dumps(out,ensure_ascii=False,indent=2));c.close()
