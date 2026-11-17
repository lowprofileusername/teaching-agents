"""Create an isolated Word fixture and verify exported OOXML, no Office required."""
import sys,zipfile,json,posixpath,xml.etree.ElementTree as ET
from pathlib import Path
W='http://schemas.openxmlformats.org/wordprocessingml/2006/main'
ns={'w':W}
if sys.argv[1]=='create':
 dest=Path(sys.argv[2]);dest.parent.mkdir(parents=True,exist_ok=True)
 # A normal paragraph spans two runs; complex paragraph preserves a drawing.
 body='<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>本课程非常重视</w:t></w:r><w:r><w:t>教学问题，建议开展讨论并保留证据。</w:t></w:r></w:p>'
 body+='<w:p><w:r><w:t>复杂段落建议保留图形。</w:t></w:r><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:extent cx="1000" cy="1000"/><wp:docPr id="1" name="Fixture drawing"/></wp:inline></w:drawing></w:r></w:p>'
 body+='<w:p><w:r><w:t>'+('本段是用于测试字数提醒的材料，不代表真实教学成果。'*190)+'</w:t></w:r></w:p>'
 parts={'[Content_Types].xml':'<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>','_rels/.rels':'<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>','word/document.xml':'<w:document xmlns:w="'+W+'"><w:body>'+body+'<w:sectPr/></w:body></w:document>'}
 with zipfile.ZipFile(dest,'w',zipfile.ZIP_DEFLATED) as z:
  for name,data in parts.items():z.writestr(name,data)
 print(dest)
else:
 src,dest=map(Path,sys.argv[2:4]);out={}
 with zipfile.ZipFile(dest) as z,zipfile.ZipFile(src) as original:
  root=ET.fromstring(z.read('word/document.xml'));ps=root.findall('.//w:body/w:p',ns)
  orig=ET.fromstring(original.read('word/document.xml')).findall('.//w:body/w:p',ns)
  deleted=[''.join(e.itertext()) for e in ps[0].findall('.//w:del/w:r/w:delText',ns)]
  inserted=[''.join(e.itertext()) for e in ps[0].findall('.//w:ins/w:r/w:t',ns)]
  assert deleted==['非常','问题'],deleted
  assert inserted==['重点','难题'],inserted
  plain=''.join(e.text or '' for e in ps[0].findall('.//w:t',ns))
  assert '建议开展讨论' in plain and '拟组织讨论' not in plain,plain
  assert ps[1].find('.//w:del',ns) is None and ps[1].find('.//w:ins',ns) is None
  assert ''.join(e.text or '' for e in ps[1].findall('.//w:t',ns))==''.join(e.text or '' for e in orig[1].findall('.//w:t',ns))
  assert ET.tostring(ps[1].find('.//w:drawing',ns))==ET.tostring(orig[1].find('.//w:drawing',ns))
  rels=ET.fromstring(z.read('word/_rels/document.xml.rels'))
  targets=[r.get('Target') for r in rels if r.get('Type','').endswith('/comments') and r.get('TargetMode')!='External']
  assert len(targets)==1,targets
  target=targets[0]
  comment_path=posixpath.normpath(target.lstrip('/') if target.startswith('/') else posixpath.join('word',target))
  comments=ET.fromstring(z.read(comment_path));ct=''.join(comments.itertext())
  for s in ['修订明细','待补','复杂段落','非常','重点']:assert s in ct,s
  assert all(c.get('{'+W+'}author')=='教创赛专家' for c in comments)
  out={'accepted_span_text_exact':True,'rejected_revision_absent':True,'complex_body_and_drawing_unchanged':True,'comments_and_revision_details':True,'author':'教创赛专家'}
 print(json.dumps(out,ensure_ascii=False))
