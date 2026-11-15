import sqlite3,sys,pathlib
p=pathlib.Path(sys.argv[1]);c=sqlite3.connect(p);c.executescript((pathlib.Path(__file__).parent/'fixture_schema.sql').read_text())
c.execute("INSERT INTO video(tag,title,teacher,institution,course,duration) VALUES('synthetic','Synthetic test lecture','测试专家','测试学校','测试课程',120)")
c.execute("INSERT INTO page(id,tag,page_no,title,page_text) VALUES('synthetic:R01','synthetic',1,'合成测试页','这是一条合成测试资料，不来自任何真实讲座。')")
c.execute("INSERT INTO page_occurrence(id,page_id,t0,t1) VALUES('occ1','synthetic:R01',0,30)")
c.execute("INSERT INTO utterance(id,tag,version,t0,t1,text) VALUES(1,'synthetic','synthetic',1,3,'合成测试转写')")
c.execute("INSERT INTO page_utterance(page_id,occurrence_id,utterance_id,overlap_start,overlap_end) VALUES('synthetic:R01','occ1',1,1,3)")
c.commit();c.close()
