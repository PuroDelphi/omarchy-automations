#!/usr/bin/env python3
"""Check native form bounds and installed light/dark themes in private profiles."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from native_ui import attach_native_ui

ROOT = Path(__file__).resolve().parents[1]
ARTIFACTS = ROOT / '.dev'
ARTIFACTS.mkdir(exist_ok=True)


def wait(check, label):
    until = time.monotonic() + 15
    while time.monotonic() < until:
        result = check()
        if result:
            return result
        time.sleep(.1)
    raise AssertionError(label)


with tempfile.TemporaryDirectory(prefix='omarchy-layout-') as temporary:
    base = Path(temporary)
    ui = base / 'ui'
    ui.mkdir()
    shutil.copytree(ROOT / 'qml', ui / 'qml')
    panel = (ROOT / 'Panel.qml').read_text().replace('minimumSize: Qt.size(760, 560)', 'minimumSize: Qt.size(0, 0)')
    panel = panel.replace('    id: root\n', '''    id: root
    function qaResize(w, h) { window.minimumSize = Qt.size(w,h); window.maximumSize = Qt.size(w,h); window.implicitWidth = w; window.implicitHeight = h; }
    function qaGeometry() {
        function rect(item) {
            var p = item.mapToItem(window.contentItem, 0, 0);
            return {x:p.x, y:p.y, width:item.width, height:item.height};
        }
        return {width:window.width, height:window.height, language:I18n.language,
            foreground:String(Color.foreground), background:String(Color.background),
            opened:editor.opened, footer:rect(editor.footer),
            save:rect(editor.footer.standardButton(Dialog.Save)),
            cancel:rect(editor.footer.standardButton(Dialog.Cancel))};
    }
    function qaHistory(enabled) { section = 4; queueOnly = enabled; }
    function qaHistoryGeometry() {
        var texts = [];
        function collect(item) {
            if (item.text !== undefined && item.truncated !== undefined)
                texts.push({text:item.text, truncated:item.truncated});
            for (var i=0; i<item.children.length; i++) collect(item.children[i]);
        }
        collect(historyFilter);
        var buttons=[];
        function bounds(item) {
            if (item.clicked !== undefined && item.text !== undefined && item.visible) {
                var at=item.mapToItem(window.contentItem,0,0);
                buttons.push({text:item.text,x:at.x,y:at.y,width:item.width,height:item.height});
            }
            for(var j=0;j<item.children.length;j++) bounds(item.children[j]);
        }
        bounds(headerActions); bounds(toolbarActions);
        var p=historyFilter.mapToItem(window.contentItem,0,0);
        var side=sidebarScroll.mapToItem(window.contentItem,0,0);
        return {x:p.x,y:p.y,width:historyFilter.width,height:historyFilter.height,texts:texts,buttons:buttons,
          sidebar:{y:side.y,height:sidebarScroll.height,contentHeight:sidebarScroll.contentHeight}};
    }
    function qaSidebarFocus(last) {
        if (last) sidebarRefresh.forceActiveFocus();
        else languageChoice.forceActiveFocus();
    }
    function qaSidebarGeometry() {
        var item=sidebarRefresh.activeFocus ? sidebarRefresh : languageChoice;
        var p=item.mapToItem(sidebarScroll,0,0);
        return {focus:item.activeFocus,y:p.y,height:item.height,viewport:sidebarScroll.availableHeight,
          contentY:sidebarScroll.contentItem.contentY};
    }
    property var qaActiveDialog: null
    function qaOpenDialog(name) {
        var dialogs={approval:approval,cancel:cancelDialog,retry:retryDialog,deleteCredential:deleteCredential,
          credential:secretDialog,simulation:sampleDialog,realTest:realTest,result:simulationResult,
          resetJournal:resetJournalDialog,detail:detailDialog,file:fileDialog};
        if (name === "oauthImport" || name === "oauthBrowser") {
            credentialType.currentIndex = name === "oauthImport" ? 1 : 2;
            qaActiveDialog=secretDialog;
        } else {
            credentialType.currentIndex=0;
            qaActiveDialog=dialogs[name] || fileDialog;
        }
        fileDialog.exporting=name === "export";
        fileDialog.diagnostic=name === "diagnostic";
        root.detailTitle="Execution details";
        qaActiveDialog.open();
    }
    function qaCaptureDialog(path) {
        qaActiveDialog.background.parent.grabToImage(function(result) {result.saveToFile(path);});
    }
    function qaCloseDialog() { qaActiveDialog.reject(); }
    function qaDialogGeometry() {
        var d=qaActiveDialog;
        var p=d.footer.mapToItem(window.contentItem,0,0);
        return {opened:d.opened,x:d.x,y:d.y,width:d.width,height:d.height,
          footer:{x:p.x,y:p.y,width:d.footer.width,height:d.footer.height}};
    }
    function qaCapture(path) {
        // A content-only grab otherwise has a transparent backdrop. Use the
        // actual dialog surface color so theme contrast can be inspected.
        editor.contentItem.background = Qt.createQmlObject('import QtQuick; Rectangle { color: "' + Color.popups.background + '" }', editor.contentItem);
        editor.contentItem.grabToImage(function(result) {result.saveToFile(path);}); editor.footer.grabToImage(function(result) {result.saveToFile(path.replace(".png", "-footer.png"));}); }
''', 1)
    (ui / 'Panel.qml').write_text(panel)
    harness = (ROOT / 'Development.qml').read_text().replace('target: "quatrro-development"', '''target: "quatrro-development"
        function resize(w: int, h: int): void { panel.qaResize(w,h); }
        function history(enabled: bool): void { panel.qaHistory(enabled); }
        function sidebarFocus(last: bool): void { panel.qaSidebarFocus(last); }
        function sidebarGeometry(): string { return JSON.stringify(panel.qaSidebarGeometry()); }
        function historyGeometry(): string { return JSON.stringify(panel.qaHistoryGeometry()); }
        function openDialog(name: string): void { panel.qaOpenDialog(name); }
        function captureDialog(path: string): void { panel.qaCaptureDialog(path); }
        function closeDialog(): void { panel.qaCloseDialog(); }
        function dialogGeometry(): string { return JSON.stringify(panel.qaDialogGeometry()); }
        function geometry(): string { return JSON.stringify(panel.qaGeometry()); }
        function captureWindow(path: string): void { panel.qaCapture(path); }''', 1)
    (ui / 'Development.qml').write_text(harness)
    checked = 0
    dialogs_checked = 0
    for theme in ('tokyo-night', 'catppuccin-latte'):
        home = base / theme
        current = home / '.local/state/omarchy/current'
        current.mkdir(parents=True)
        (current / 'theme').symlink_to(Path('/usr/share/omarchy/themes') / theme, target_is_directory=True)
        env = dict(os.environ, HOME=str(home), QUATRRO_PROFILE=str(home / 'profile'),
                   PATH=str(ROOT / 'build') + os.pathsep + os.environ['PATH'],
                   QT_QPA_PLATFORM='offscreen', QT_QUICK_BACKEND='software')
        attach_native_ui(env, home)
        log_path = ARTIFACTS / ('layout-' + theme + '.log')
        children = []
        def ipc(method, *args):
            return subprocess.check_output(['quickshell','ipc','-p',str(ui / 'Development.qml'),
                                            'call','quatrro-development',method,*args],
                                           env=env,text=True,stderr=subprocess.DEVNULL,timeout=10).strip()
        def state():
            try: return json.loads(ipc('state'))
            except (subprocess.CalledProcessError, json.JSONDecodeError): return {}
        try:
            with log_path.open('w') as log:
                children.append(subprocess.Popen([ROOT / 'build/quatrrod','--listen='],env=env,stdout=log,stderr=log))
                wait(lambda: (home / 'profile/runtime/control.sock').exists(), 'engine')
                children.append(subprocess.Popen(['quickshell','-p',str(ui / 'Development.qml')],env=env,stdout=log,stderr=log))
                wait(lambda: state().get('loaded'), 'panel')
                for language,width,height in [('en',1100,800),('es',664,718),('en',540,600),('es',540,600)]:
                    ipc('language',language)
                    wait(lambda: state().get('language') == language and not state().get('busy'), 'language')
                    ipc('resize',str(width),str(height))
                    wait(lambda: g if (g := json.loads(ipc('geometry')))['width']==width and g['height']==height else None, 'history viewport resize')
                    for pending in ('false', 'true'):
                        ipc('history',pending)
                        time.sleep(.15)
                        history=json.loads(ipc('historyGeometry'))
                        assert history['sidebar']['y']+history['sidebar']['height'] <= height+1, history
                        assert len(history['buttons']) == 7, history
                        for button in history['buttons']:
                            assert button['x']>=0 and button['y']>=0 and button['x']+button['width']<=width+1 and button['y']+button['height']<=height+1, (theme,language,width,button)
                        assert history['width']>0 and history['height']>0, history
                        assert history['x']>=0 and history['x']+history['width']<=width+1, history
                        assert history['y']>=0 and history['y']+history['height']<=height+1, history
                        assert any(row['text'] for row in history['texts']), history
                        assert not any(row['truncated'] for row in history['texts']), (theme,language,width,history)
                    for last in ('true','false'):
                        ipc('sidebarFocus',last)
                        time.sleep(.1)
                        side=json.loads(ipc('sidebarGeometry'))
                        assert side['focus'] and side['y']>=-1 and side['y']+side['height']<=side['viewport']+1, (theme,language,width,side)
                    history_capture=ARTIFACTS/f'layout-{theme}-{language}-{width}-history.png'
                    history_capture.unlink(missing_ok=True)
                    ipc('capture',str(history_capture))
                    wait(history_capture.exists,'history capture')
                    for kind in ('entries','destinations','actions','flows','monitors','timers','scripts','adapters'):
                        ipc('openEditor',kind)
                        geometry = wait(lambda: g if (g := json.loads(ipc('geometry')))['opened'] and g['width']==width and g['height']==height else None, f'dialog layout {theme} {language} {width}x{height} {kind}: '+ipc('geometry'))
                        for name in ('footer','save','cancel'):
                            rect=geometry[name]
                            assert rect['width']>0 and rect['height']>0, (theme,language,kind,geometry)
                            assert rect['x']>=0 and rect['y']>=0 and rect['x']+rect['width']<=width+1 and rect['y']+rect['height']<=height+1, (theme,language,kind,geometry)
                        if kind in ('entries','actions'):
                            capture=ARTIFACTS/f'layout-{theme}-{language}-{width}-{kind}.png'
                            capture.unlink(missing_ok=True)
                            ipc('captureWindow',str(capture))
                            wait(lambda: capture.exists() and capture.with_name(capture.stem+'-footer.png').exists(),'form and footer captures')
                        ipc('dismissEditor')
                        checked += 1
                    for dialog in ('approval','cancel','retry','deleteCredential','credential','simulation','realTest','result','resetJournal','detail','file','oauthImport','oauthBrowser','export','diagnostic'):
                        ipc('openDialog',dialog)
                        dg=wait(lambda: g if (g:=json.loads(ipc('dialogGeometry')))['opened'] else None, 'dialog '+dialog)
                        for rectangle in (dg,dg['footer']):
                            assert rectangle['width']>0 and rectangle['height']>0, (dialog,dg)
                            assert rectangle['x']>=0 and rectangle['y']>=0 and rectangle['x']+rectangle['width']<=width+1 and rectangle['y']+rectangle['height']<=height+1, (theme,language,width,dialog,dg)
                        if width == 664 or (width == 540 and language == 'en'):
                            dialog_capture=ARTIFACTS/f'dialog-{theme}-{language}-{width}-{dialog}.png'
                            dialog_capture.unlink(missing_ok=True)
                            ipc('captureDialog',str(dialog_capture))
                            wait(dialog_capture.exists,'dialog capture '+dialog)
                        ipc('closeDialog')
                        dialogs_checked += 1
                assert geometry['background'] == ('#1a1b26' if theme=='tokyo-night' else '#eff1f5'), geometry
        finally:
            for child in reversed(children):
                child.terminate()
                try: child.wait(timeout=5)
                except subprocess.TimeoutExpired: child.kill();child.wait()
        output=log_path.read_text()
        assert not any(term in output for term in ('ERROR:', 'TypeError:', 'ReferenceError:')), output
    print(json.dumps(dict(form_layouts=checked, dialog_layouts=dialogs_checked, installed_themes=['tokyo-night','catppuccin-latte'],
                          languages=['en','es'], sizes=[[1100,800],[664,718],[540,600]], host_theme_unchanged=True)))
