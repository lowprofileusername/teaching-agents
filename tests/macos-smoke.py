from pathlib import Path
import os,subprocess,tempfile,urllib.request,json
root=Path(os.environ.get("SMOKE_DIR","dist/mac-smoke"));root.mkdir(parents=True,exist_ok=True)
# Use a synthetic fixture; never upload or depend on private lecture data.
fixture=root/"fixture.sqlite3"
if fixture.exists():fixture.unlink()
subprocess.run(["python3","tests/make_fixture.py",str(fixture)],check=True)
for app in Path(os.environ.get("MAC_DIST","dist/macos")).glob("*.app"):
 env=os.environ.copy();env["TEACHING_LIBRARY_DB"]=str((root/"fixture.sqlite3").resolve());env["TEACHING_AGENT_PROFILES"]=str((root/"profiles.json").resolve())
 p=subprocess.Popen([str(app/"Contents/MacOS/TeachingAgent"),"-no-browser"],stdout=subprocess.PIPE,text=True,env=env)
 try:
  link=p.stdout.readline().strip();base,token=link.split("/#")
  def post(route,data):
   req=urllib.request.Request(base+route,data=json.dumps(data).encode(),headers={"X-Session-Token":token,"Content-Type":"application/json"});return json.load(urllib.request.urlopen(req,timeout=20))
  assert post("/api/library",{})["counts"]["video"]==1
  assert post("/api/search",{"q":"测试"})["pages"]
  post("/api/quit",{});p.wait(timeout=10)
  print(app.name,"launch and SQLite smoke passed")
 finally:
  if p.poll() is None:p.terminate()
