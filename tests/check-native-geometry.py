import json,sqlite3,sys,time
from pathlib import Path
p=Path(sys.argv[1])
# The real native window was moved/resized by Win32 before this check.
last=None
for _ in range(20):
 try:
  with sqlite3.connect(f'{p.as_uri()}?mode=ro',uri=True,timeout=1) as db:
   row=db.execute("select geometry from window_layout where name='workbench'").fetchone()
  if row:
   g=json.loads(row[0]);last=g
   if abs(g['x']-100)<=12 and abs(g['y']-50)<=12 and 800<=g['width']<=1020 and 600<=g['height']<=680:
    print('PASS native move/resize persisted:',g);break
 except (sqlite3.Error,OSError):pass
 time.sleep(.25)
else:raise AssertionError(f'Native window rectangle not saved correctly: {last}')
