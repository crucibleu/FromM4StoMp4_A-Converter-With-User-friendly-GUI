// m4s-converter 图形界面版本
package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"m4s-converter/common"
	"m4s-converter/internal"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	"github.com/sirupsen/logrus"
)

//go:embed translations
var translations embed.FS

//go:embed Icon.png
var iconPNG []byte

const (
	prefCachePath     = "cachePath"
	prefDanmaku       = "danmaku"
	prefOverlay       = "overlay"
	prefSummarize     = "summarize"
	prefCustomMP4Box  = "customMP4Box"
	prefMP4BoxPath    = "mp4boxPath"
	prefAcceptedTerms = "acceptedTerms"
)

func main() {
	a := app.NewWithID("io.github.mzky.m4s-converter")
	_ = lang.AddTranslationsFS(translations, "translations")
	a.SetIcon(fyne.NewStaticResource("Icon.png", iconPNG))

	w := a.NewWindow("m4s-converter")
	u := newUI(a, w)
	setupLogging(u.log)

	w.SetContent(u.build())
	w.Resize(fyne.NewSize(800, 720))
	w.CenterOnScreen()
	w.SetCloseIntercept(u.onClose)
	u.showTermsIfNeeded()
	w.ShowAndRun()
}

// setupLogging 将合成过程的日志同时写入界面与日志文件
func setupLogging(view *logView) {
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true, FullTimestamp: true, TimestampFormat: "2006-01-02 15:04:05"})
	logrus.SetOutput(io.Discard)
	if dir, err := os.UserCacheDir(); err == nil {
		dir = filepath.Join(dir, "m4s-converter")
		if os.MkdirAll(dir, 0o755) == nil {
			if f, err := os.Create(filepath.Join(dir, "m4s-gui.log")); err == nil {
				logrus.SetOutput(f)
			}
		}
	}
	logrus.AddHook(view)
}

type ui struct {
	app  fyne.App
	win  fyne.Window
	pref fyne.Preferences

	cacheEntry   *widget.Entry
	outputLabel  *widget.Label
	danmaku      *widget.Check
	overlay      *widget.Check
	summarize    *widget.Check
	customMP4Box *widget.Check
	mp4boxEntry  *widget.Entry
	mp4boxBrowse *widget.Button
	cacheBrowse  *widget.Button

	startBtn  *widget.Button
	cancelBtn *widget.Button
	openBtn   *widget.Button
	bar       *widget.ProgressBar
	barBusy   *widget.ProgressBarInfinite
	status    *widget.Label
	log       *logView

	mu      sync.Mutex
	running *common.Config // 正在执行的任务，nil 表示空闲
	lastOut string
}

func newUI(a fyne.App, w fyne.Window) *ui {
	return &ui{app: a, win: w, pref: a.Preferences(), log: newLogView()}
}

func (u *ui) build() fyne.CanvasObject {
	u.cacheEntry = widget.NewEntry()
	u.cacheEntry.SetPlaceHolder(lang.X("cache.placeholder", "Folder that contains the Bilibili cached videos"))
	u.cacheEntry.SetText(u.pref.StringWithFallback(prefCachePath, common.DefaultCachePath()))
	u.outputLabel = widget.NewLabel("")
	u.outputLabel.Truncation = fyne.TextTruncateEllipsis
	u.cacheEntry.OnChanged = func(string) { u.updateOutputLabel() }
	u.updateOutputLabel()
	u.cacheBrowse = widget.NewButtonWithIcon(lang.X("browse", "Browse…"), theme.FolderOpenIcon(), func() {
		u.pickFolder(lang.X("cache.pick", "Choose the Bilibili cache folder"), u.cacheEntry)
	})

	u.danmaku = widget.NewCheck(lang.X("opt.danmaku", "Generate danmaku (bullet comments) subtitle file (.ass)"), nil)
	u.danmaku.SetChecked(u.pref.BoolWithFallback(prefDanmaku, true))
	u.overlay = widget.NewCheck(lang.X("opt.overlay", "Overwrite videos that were already converted"), nil)
	u.overlay.SetChecked(u.pref.Bool(prefOverlay))
	u.summarize = widget.NewCheck(lang.X("opt.summarize", "Also copy the separate audio/video files into the output folder"), nil)
	u.summarize.SetChecked(u.pref.Bool(prefSummarize))

	u.mp4boxEntry = widget.NewEntry()
	u.mp4boxEntry.SetPlaceHolder(lang.X("mp4box.placeholder", "Path to MP4Box executable"))
	u.mp4boxEntry.SetText(u.pref.String(prefMP4BoxPath))
	u.mp4boxBrowse = widget.NewButtonWithIcon(lang.X("browse", "Browse…"), theme.FileIcon(), u.pickMP4Box)
	u.customMP4Box = widget.NewCheck(lang.X("mp4box.custom", "Use my own MP4Box (GPAC) instead of the built-in one"), func(on bool) {
		u.setEnabled(u.mp4boxEntry, on)
		u.setEnabled(u.mp4boxBrowse, on)
	})
	u.customMP4Box.SetChecked(u.pref.Bool(prefCustomMP4Box))
	u.customMP4Box.OnChanged(u.customMP4Box.Checked)

	u.startBtn = widget.NewButtonWithIcon(lang.X("start", "Start conversion"), theme.MediaPlayIcon(), u.start)
	u.startBtn.Importance = widget.HighImportance
	u.cancelBtn = widget.NewButtonWithIcon(lang.X("cancel", "Cancel"), theme.CancelIcon(), u.cancel)
	u.cancelBtn.Disable()
	u.openBtn = widget.NewButtonWithIcon(lang.X("open", "Open output folder"), theme.FolderIcon(), func() {
		dir := u.lastOut
		if dir == "" {
			dir = filepath.Join(strings.TrimSpace(u.cacheEntry.Text), "output")
		}
		if _, err := os.Stat(dir); err != nil {
			dialog.ShowInformation(lang.X("open", "Open output folder"), lang.X("open.missing", "The output folder does not exist yet. Run a conversion first."), u.win)
			return
		}
		common.OpenFolder(dir)
	})

	u.bar = widget.NewProgressBar()
	u.barBusy = widget.NewProgressBarInfinite()
	u.barBusy.Stop()
	u.barBusy.Hide()
	u.status = widget.NewLabel(lang.X("status.ready", "Ready."))
	u.status.Truncation = fyne.TextTruncateEllipsis

	cacheRow := container.NewBorder(nil, nil, nil, u.cacheBrowse, u.cacheEntry)
	mp4boxRow := container.NewBorder(nil, nil, nil, u.mp4boxBrowse, u.mp4boxEntry)
	advanced := widget.NewAccordion(widget.NewAccordionItem(lang.X("advanced", "More options"),
		container.NewVBox(u.overlay, u.summarize, u.customMP4Box, mp4boxRow)))

	form := container.NewVBox(
		widget.NewLabelWithStyle(lang.X("cache.title", "Bilibili cache folder"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		cacheRow,
		u.outputLabel,
		widget.NewSeparator(),
		u.danmaku,
		advanced,
		widget.NewSeparator(),
		container.NewBorder(nil, nil, container.NewHBox(u.startBtn, u.cancelBtn), u.openBtn),
		container.NewStack(u.bar, u.barBusy),
		u.status,
	)

	notice := widget.NewLabelWithStyle(lang.X("terms.short",
		"Only convert videos you cached yourself with the official Bilibili client, for personal backup only."),
		fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	notice.Wrapping = fyne.TextWrapWord

	logCard := container.NewBorder(
		container.NewBorder(nil, nil,
			widget.NewLabelWithStyle(lang.X("log", "Log"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewButtonWithIcon(lang.X("log.copy", "Copy"), theme.ContentCopyIcon(), func() {
				u.app.Clipboard().SetContent(u.log.text())
			})),
		nil, nil, nil, u.log.scroll)

	return container.NewBorder(container.NewPadded(form), container.NewPadded(notice), nil, nil, container.NewPadded(logCard))
}

func (u *ui) updateOutputLabel() {
	p := strings.TrimSpace(u.cacheEntry.Text)
	if p == "" {
		u.outputLabel.SetText("")
		return
	}
	u.outputLabel.SetText(lang.X("output.where", "Videos will be saved to: {{.Dir}}", map[string]any{"Dir": filepath.Join(p, "output")}))
}

func (u *ui) setEnabled(w fyne.Disableable, on bool) {
	if on {
		w.Enable()
	} else {
		w.Disable()
	}
}

// pickFolder 优先使用系统原生对话框，不可用时（如 Linux 未安装 zenity）回退到 Fyne 内置对话框
func (u *ui) pickFolder(title string, target *widget.Entry) {
	start := strings.TrimSpace(target.Text)
	go func() {
		p, err := zenity.SelectFile(zenity.Title(title), zenity.Directory(), zenity.Filename(start))
		fyne.Do(func() {
			switch {
			case err == nil:
				target.SetText(p)
			case err == zenity.ErrCanceled:
			default:
				d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
					if err == nil && uri != nil {
						target.SetText(uri.Path())
					}
				}, u.win)
				d.Resize(fyne.NewSize(700, 500))
				d.Show()
			}
		})
	}()
}

func (u *ui) pickMP4Box() {
	start := strings.TrimSpace(u.mp4boxEntry.Text)
	go func() {
		p, err := zenity.SelectFile(zenity.Title(lang.X("mp4box.pick", "Choose the MP4Box executable")), zenity.Filename(start))
		fyne.Do(func() {
			switch {
			case err == nil:
				u.mp4boxEntry.SetText(p)
			case err == zenity.ErrCanceled:
			default:
				d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
					if err == nil && r != nil {
						u.mp4boxEntry.SetText(r.URI().Path())
						_ = r.Close()
					}
				}, u.win)
				d.Resize(fyne.NewSize(700, 500))
				d.Show()
			}
		})
	}()
}

func (u *ui) savePrefs() {
	u.pref.SetString(prefCachePath, strings.TrimSpace(u.cacheEntry.Text))
	u.pref.SetBool(prefDanmaku, u.danmaku.Checked)
	u.pref.SetBool(prefOverlay, u.overlay.Checked)
	u.pref.SetBool(prefSummarize, u.summarize.Checked)
	u.pref.SetBool(prefCustomMP4Box, u.customMP4Box.Checked)
	u.pref.SetString(prefMP4BoxPath, strings.TrimSpace(u.mp4boxEntry.Text))
}

func (u *ui) showError(msg string) {
	dialog.ShowError(fmt.Errorf("%s", msg), u.win)
}

func (u *ui) start() {
	cachePath := strings.TrimSpace(u.cacheEntry.Text)
	if st, err := os.Stat(cachePath); cachePath == "" || err != nil || !st.IsDir() {
		u.showError(lang.X("err.nofolder", "Please choose an existing Bilibili cache folder."))
		return
	}
	c := &common.Config{
		CachePath: cachePath,
		AssOFF:    !u.danmaku.Checked,
		Overlay:   u.overlay.Checked,
		Summarize: u.summarize.Checked,
	}
	if u.customMP4Box.Checked {
		c.GPACPath = strings.TrimSpace(u.mp4boxEntry.Text)
		if st, err := os.Stat(c.GPACPath); c.GPACPath == "" || err != nil || st.IsDir() {
			u.showError(lang.X("err.mp4box", "The MP4Box file you chose does not exist."))
			return
		}
	}
	c.OnProgress = func(stage common.Stage, done, total int, name string) {
		fyne.Do(func() { u.onProgress(stage, done, total, name) })
	}
	u.savePrefs()
	u.log.clear()
	u.setRunning(c)
	go u.run(c)
}

func (u *ui) run(c *common.Config) {
	var res *common.Result
	var runErr error
	defer func() {
		if r := recover(); r != nil {
			runErr = fmt.Errorf("%v", r)
			logrus.Error("程序异常: ", r)
		}
		fyne.Do(func() { u.finish(c, res, runErr) })
	}()

	logrus.Info("BiliBili 缓存目录: ", c.CachePath)
	if err := c.CheckCachePath(); err != nil {
		runErr = fmt.Errorf("%s", lang.X("err.nom4s", "No .m4s files were found in this folder. Choose the folder where the Bilibili client stores its cache (usually Videos/bilibili)."))
		return
	}
	if c.GPACPath == "" {
		p, err := internal.GetMP4Box()
		if err != nil {
			runErr = err
			return
		}
		c.GPACPath = p
	}
	logrus.Info("使用MP4Box进行音视频合成: ", c.GPACPath)
	res, runErr = c.Convert()
}

func (u *ui) onProgress(stage common.Stage, done, total int, name string) {
	if c := u.current(); c != nil && c.ShouldExit() {
		return // 保持“正在取消”的提示
	}
	switch stage {
	case common.StageExtract:
		u.showBusyBar(true)
		if name != "" {
			u.status.SetText(lang.X("status.extract", "Preparing: {{.Name}}", map[string]any{"Name": name}))
		} else {
			u.status.SetText(lang.X("status.scan", "Scanning the cache folder…"))
		}
	case common.StageMerge:
		u.showBusyBar(false)
		if total > 0 {
			u.bar.SetValue(float64(done) / float64(total))
		}
		if done < total {
			u.status.SetText(lang.X("status.merge", "Merging folder {{.Done}} of {{.Total}}…", map[string]any{"Done": done + 1, "Total": total}))
		}
	}
}

func (u *ui) showBusyBar(busy bool) {
	if busy {
		u.bar.Hide()
		u.barBusy.Show()
		u.barBusy.Start()
	} else {
		u.barBusy.Stop()
		u.barBusy.Hide()
		u.bar.Show()
	}
}

func (u *ui) current() *common.Config {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.running
}

func (u *ui) setRunning(c *common.Config) {
	u.mu.Lock()
	u.running = c
	u.mu.Unlock()

	busy := c != nil
	for _, w := range []fyne.Disableable{u.cacheEntry, u.cacheBrowse, u.danmaku, u.overlay, u.summarize, u.customMP4Box, u.startBtn} {
		u.setEnabled(w, !busy)
	}
	u.setEnabled(u.mp4boxEntry, !busy && u.customMP4Box.Checked)
	u.setEnabled(u.mp4boxBrowse, !busy && u.customMP4Box.Checked)
	u.setEnabled(u.cancelBtn, busy)
	if busy {
		u.bar.SetValue(0)
		u.showBusyBar(true)
		u.status.SetText(lang.X("status.scan", "Scanning the cache folder…"))
	} else {
		u.showBusyBar(false)
	}
}

func (u *ui) cancel() {
	if c := u.current(); c != nil {
		c.SetExitFlag(true)
		u.cancelBtn.Disable()
		u.status.SetText(lang.X("status.canceling", "Canceling — waiting for the current file to finish…"))
	}
}

func (u *ui) finish(c *common.Config, res *common.Result, err error) {
	u.setRunning(nil)
	u.log.flush()
	if err != nil {
		u.bar.SetValue(0)
		u.status.SetText(lang.X("status.failed", "Failed."))
		u.showError(err.Error())
		return
	}
	u.lastOut = res.OutputDir
	u.bar.SetValue(1)

	summary := lang.X("done.summary", "Converted {{.Count}} video(s) in {{.Secs}} s.", map[string]any{"Count": len(res.OutputFiles), "Secs": res.ElapsedSecs})
	var extra []string
	if res.Canceled {
		summary = lang.X("done.canceled", "Canceled. {{.Count}} video(s) were converted before stopping.", map[string]any{"Count": len(res.OutputFiles)})
	}
	if n := len(res.SkippedDirs); n > 0 {
		extra = append(extra, lang.X("done.skipped", "{{.Count}} video(s) skipped because the download was not finished.", map[string]any{"Count": n}))
	}
	if n := len(res.FailedFiles); n > 0 {
		extra = append(extra, lang.X("done.failed", "{{.Count}} video(s) failed — see the log for details.", map[string]any{"Count": n}))
	}
	if len(res.OutputFiles) == 0 && !res.Canceled {
		extra = append(extra, lang.X("done.none", "No new videos were converted (videos that were already converted are skipped)."))
	}
	u.status.SetText(summary)

	msg := widget.NewLabel(strings.Join(append([]string{summary}, extra...), "\n"))
	msg.Wrapping = fyne.TextWrapWord
	if len(res.OutputFiles) == 0 {
		dialog.ShowCustom(lang.X("done.title", "Finished"), lang.X("ok", "OK"), msg, u.win)
		return
	}
	d := dialog.NewCustomConfirm(lang.X("done.title", "Finished"), lang.X("open", "Open output folder"), lang.X("close", "Close"), msg, func(open bool) {
		if open {
			common.OpenFolder(res.OutputDir)
		}
	}, u.win)
	d.Resize(fyne.NewSize(460, 0))
	d.Show()
}

func (u *ui) onClose() {
	c := u.current()
	if c == nil {
		u.savePrefs()
		u.win.Close()
		return
	}
	dialog.ShowConfirm(lang.X("quit.title", "Conversion in progress"),
		lang.X("quit.msg", "A conversion is still running. Stop it and quit?"), func(ok bool) {
			if ok {
				c.SetExitFlag(true)
				u.savePrefs()
				u.win.Close()
			}
		}, u.win)
}

func (u *ui) showTermsIfNeeded() {
	if u.pref.Bool(prefAcceptedTerms) {
		return
	}
	text := widget.NewLabel(lang.X("terms.full",
		"By using this program you agree to the following terms:\n\n"+
			"Only convert videos that you cached yourself through the official Bilibili client, "+
			"and use the converted files strictly as a personal backup. "+
			"Do not share, distribute or use them commercially."))
	text.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm(lang.X("terms.title", "Terms of use"), lang.X("terms.agree", "I agree"), lang.X("terms.quit", "Quit"), text, func(ok bool) {
		if !ok {
			u.app.Quit()
			return
		}
		u.pref.SetBool(prefAcceptedTerms, true)
		logrus.Info("用户同意使用，程序继续执行")
	}, u.win)
	d.Resize(fyne.NewSize(480, 0))
	d.Show()
}

// logView 显示合成日志；日志先写入缓冲区，再定时批量刷新到界面，避免大量日志拖慢界面
type logView struct {
	label  *widget.Label
	scroll *container.Scroll

	mu      sync.Mutex
	pending []string
	lines   []string
}

const maxLogLines = 3000

func newLogView() *logView {
	l := &logView{label: widget.NewLabel("")}
	l.label.Wrapping = fyne.TextWrapWord
	l.label.Selectable = true
	l.scroll = container.NewVScroll(l.label)
	l.scroll.SetMinSize(fyne.NewSize(0, 160))
	go func() {
		for range time.Tick(200 * time.Millisecond) {
			fyne.Do(l.flush)
		}
	}()
	return l
}

// Levels 实现 logrus.Hook
func (l *logView) Levels() []logrus.Level { return logrus.AllLevels }

// Fire 实现 logrus.Hook
func (l *logView) Fire(e *logrus.Entry) error {
	prefix := ""
	switch e.Level {
	case logrus.WarnLevel:
		prefix = "⚠ "
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		prefix = "✖ "
	}
	line := e.Time.Format("15:04:05") + "  " + prefix + strings.TrimRight(e.Message, "\n")
	l.mu.Lock()
	l.pending = append(l.pending, line)
	l.mu.Unlock()
	return nil
}

// flush 必须在界面线程中调用
func (l *logView) flush() {
	l.mu.Lock()
	if len(l.pending) == 0 {
		l.mu.Unlock()
		return
	}
	l.lines = append(l.lines, l.pending...)
	l.pending = nil
	if len(l.lines) > maxLogLines {
		l.lines = l.lines[len(l.lines)-maxLogLines:]
	}
	text := strings.Join(l.lines, "\n")
	l.mu.Unlock()
	l.label.SetText(text)
	l.scroll.ScrollToBottom()
}

func (l *logView) clear() {
	l.mu.Lock()
	l.lines, l.pending = nil, nil
	l.mu.Unlock()
	l.label.SetText("")
}

func (l *logView) text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(append(append([]string{}, l.lines...), l.pending...), "\n")
}
