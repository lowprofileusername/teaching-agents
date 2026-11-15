from pathlib import Path
import os,subprocess,plistlib,hashlib
out=Path(os.environ.get("MAC_DIST","dist/macos"));out.mkdir(parents=True,exist_ok=True)
for persona,label in [("lijing","李静教学资料助手"),("original","原创女教师教学助手"),("male","原创男教师教学助手")]:
 app=out/(label+".app");exe=app/"Contents/MacOS/TeachingAgent";exe.parent.mkdir(parents=True,exist_ok=True)
 env=os.environ.copy();env.update(CGO_ENABLED="1",GOOS="darwin",GOARCH="arm64",MACOSX_DEPLOYMENT_TARGET="15.0",CGO_CFLAGS="-mmacosx-version-min=15.0",CGO_LDFLAGS="-mmacosx-version-min=15.0")
 subprocess.run([os.environ.get("GO_BIN","go"),"build","-trimpath","-ldflags",f"-s -w -X main.defaultPersona={persona}","-o",str(exe),"."],check=True,env=env)
 info={"CFBundleName":label,"CFBundleDisplayName":label,"CFBundleIdentifier":"org.teachingarchive.agent."+persona,"CFBundleVersion":"0.3.0","CFBundleShortVersionString":"0.3.0","CFBundlePackageType":"APPL","CFBundleExecutable":"TeachingAgent","LSMinimumSystemVersion":"15.0","LSUIElement":True,"NSHighResolutionCapable":True}
 (app/"Contents/Info.plist").write_bytes(plistlib.dumps(info));subprocess.run(["xattr","-cr",str(app)],check=True);subprocess.run(["codesign","--force","--sign","-",str(app)],check=True);subprocess.run(["codesign","--verify","--deep","--strict",str(app)],check=True)
 subprocess.run(["ditto","-c","-k","--sequesterRsrc","--keepParent",str(app),str(out/f"teaching-agent-{persona}-macos-arm64.zip")],check=True)
 print(label,"built and ad-hoc signed",flush=True)
with (out/"SHA256SUMS-macos.txt").open("w") as f:
 for p in out.glob("*.zip"):f.write(hashlib.sha256(p.read_bytes()).hexdigest()+"  "+p.name+"\n")
