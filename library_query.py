#!/usr/bin/env python3
"""Portable read-only SQLite interface; Python 3 standard library only."""
import argparse,sqlite3,json,time,zlib,hashlib
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--db',required=True);s=p.add_subparsers(dest='action',required=True)
s.add_parser('schema');q=s.add_parser('search');q.add_argument('--keyword',action='append',required=True);q.add_argument('--tag',default='');q=s.add_parser('page');q.add_argument('id');q=s.add_parser('export');q.add_argument('--kind',choices=['pdf','asset'],required=True);q.add_argument('--id',required=True);q.add_argument('--out',required=True)
a=p.parse_args();dbpath=Path(a.db).resolve();c=sqlite3.connect(dbpath.as_uri()+'?mode=ro',uri=True,timeout=3);c.row_factory=sqlite3.Row;c.execute('pragma query_only=on');c.execute('pragma temp_store=memory')
deadline=time.monotonic()+5;c.set_progress_handler(lambda:time.monotonic()>deadline,10000)
def rows(sql,args=()):return [dict(x) for x in c.execute(sql,args)]
claude='speaker' in {r['name'] for r in c.execute('pragma table_info(video)')}
if claude and a.action!='schema':
 if a.action=='search':
  terms=[x.strip() for x in a.keyword if x.strip()][:8]
  if not terms:raise SystemExit('Need keywords')
  pred=" OR ".join(["instr(coalesce(p.title,'')||coalesce(p.ocr_text,''),?)>0"]*len(terms))
  out={'format':'claude','pages':rows("SELECT p.tag||':P'||p.page_no AS id,p.tag,p.page_no,p.title,p.ocr_text,v.speaker,v.school,v.course FROM page p JOIN video v USING(tag) WHERE (?='' OR p.tag=?) AND ("+pred+") LIMIT 12",[a.tag,a.tag]+terms)}
 elif a.action=='page':
  tag,sep,num=a.id.rpartition(':P')
  if not sep or not num.isdigit():raise SystemExit('Claude page id example: song:P12')
  out={'format':'claude','page_number_note':'P is database order, not original PPT corner number','page':rows('SELECT tag,page_no,title,t_start,t_end,time_ranges,ocr_text,main_message,elements,function,relation,visual,evidence,borrow,migrate FROM page WHERE tag=? AND page_no=?',(tag,int(num))),'utterances':rows('SELECT t_start,t_end,text FROM utterance WHERE tag=? AND page_no=? ORDER BY t_start',(tag,int(num))),'errata':rows('SELECT * FROM errata WHERE tag=?',(tag,))}
 else:
  c.set_progress_handler(None,0)
  parts=a.id.split(':',2)
  kinds={'page-image':('page','image','id'),'frame-image':('frame','image','id'),'flash-image':('flash','image','id'),'video-audio':('video','audio','tag')}
  if a.kind!='asset' or len(parts)!=3 or parts[0]!='claude' or parts[1] not in kinds:raise SystemExit('Claude media id example: claude:page-image:202 or claude:video-audio:song; no embedded report PDFs')
  table,col,key=kinds[parts[1]];r=c.execute('SELECT '+col+' FROM '+table+' WHERE '+key+'=?',(parts[2],)).fetchone()
  if not r or not r[0]:raise SystemExit('Media not found')
  dest=Path(a.out)
  if dest.exists():raise SystemExit('Output exists; choose a new filename')
  dest.write_bytes(r[0]);out={'file':str(dest.resolve()),'bytes':len(r[0]),'sha256':hashlib.sha256(r[0]).hexdigest()}
 print(json.dumps(out,ensure_ascii=False,indent=2));c.close();raise SystemExit(0)
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
