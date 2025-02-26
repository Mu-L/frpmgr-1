package ui

import (
	"github.com/koho/frpmgr/i18n"
	"github.com/koho/frpmgr/pkg/consts"
	"github.com/koho/frpmgr/pkg/util"
	"github.com/koho/frpmgr/services"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/thoas/go-funk"
	"golang.org/x/sys/windows"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// View is the interface that must be implemented to build a Widget.
type View interface {
	// View should define widget in declarative way, and will
	// be called by the parent widget.
	View() Widget
	// OnCreate will be called after the creation of views. The
	// view reference should be available now.
	OnCreate()
	// Invalidate should be called if data that view relying on
	// is changed. The view should be updated with new data.
	Invalidate()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type FRPManager struct {
	*walk.MainWindow

	tabs      *walk.TabWidget
	confPage  *ConfPage
	logPage   *LogPage
	aboutPage *AboutPage
}

func RunUI() error {
	// Make sure the config directory exists.
	if err := os.MkdirAll(PathOfConf(""), os.ModePerm); err != nil {
		return err
	}
	if err := loadAllConfs(); err != nil {
		return err
	}
	fm := new(FRPManager)
	fm.confPage = NewConfPage()
	fm.logPage = NewLogPage()
	fm.aboutPage = NewAboutPage()
	mw := MainWindow{
		Icon:       loadLogoIcon(32),
		AssignTo:   &fm.MainWindow,
		Title:      i18n.Sprintf("FRP Manager"),
		Persistent: true,
		Visible:    false,
		Layout:     VBox{Margins: Margins{5, 5, 5, 5}},
		Font:       consts.TextRegular,
		Children: []Widget{
			TabWidget{
				AssignTo: &fm.tabs,
				Pages: []TabPage{
					fm.confPage.Page(),
					fm.logPage.Page(),
					fm.aboutPage.Page(),
				},
			},
		},
		Functions: map[string]func(args ...interface{}) (interface{}, error){
			"sysIcon": func(args ...interface{}) (interface{}, error) {
				for _, index := range args[2:] {
					if icon := loadSysIcon(args[0].(string), int32(index.(float64)), int(args[1].(float64))); icon != nil {
						return icon, nil
					}
				}
				return nil, nil
			},
		},
		OnDropFiles: fm.confPage.confView.ImportFiles,
	}
	if err := mw.Create(); err != nil {
		return err
	}
	// Initialize child pages
	fm.confPage.OnCreate()
	fm.logPage.OnCreate()
	fm.aboutPage.OnCreate()
	// Resize window
	fm.SetSizePixels(walk.Size{
		fm.confPage.confView.MinSizePixels().Width + fm.IntFrom96DPI(685),
		fm.IntFrom96DPI(525),
	})
	fm.SetVisible(true)
	fm.Run()
	services.Cleanup()
	return nil
}

func showError(err error, owner walk.Form) bool {
	if err == nil {
		return false
	}
	showErrorMessage(owner, "", err.Error())
	return true
}

func showErrorMessage(owner walk.Form, title, message string) {
	if title == "" {
		title = i18n.Sprintf("Error")
	}
	walk.MsgBox(owner, title, message, walk.MsgBoxIconError)
}

func showWarningMessage(owner walk.Form, title, message string) {
	walk.MsgBox(owner, title, message, walk.MsgBoxIconWarning)
}

func showInfoMessage(owner walk.Form, title, message string) {
	walk.MsgBox(owner, title, message, walk.MsgBoxIconInformation)
}

// openPath opens a file or url with default application
func openPath(path string) {
	if path == "" {
		return
	}
	win.ShellExecute(0, nil, windows.StringToUTF16Ptr(path), nil, nil, win.SW_SHOWNORMAL)
}

// openFolder opens the explorer and select the given file
func openFolder(path string) {
	if path == "" {
		return
	}
	if absPath, err := filepath.Abs(path); err == nil {
		win.ShellExecute(0, nil, windows.StringToUTF16Ptr(`explorer`),
			windows.StringToUTF16Ptr(`/select,`+absPath), nil, win.SW_SHOWNORMAL)
	}
}

// openFileDialog shows a file dialog to choose file or directory and sends the selected path to the LineEdit view
func openFileDialog(receiver *walk.LineEdit, title string, filter string, file bool) error {
	dlg := walk.FileDialog{
		Filter: filter + consts.FilterAllFiles,
		Title:  title,
	}
	var ok bool
	var err error
	if file {
		ok, err = dlg.ShowOpen(receiver.Form())
	} else {
		ok, err = dlg.ShowBrowseFolder(receiver.Form())
	}
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return receiver.SetText(strings.ReplaceAll(dlg.FilePath, "\\", "/"))
}

// calculateHeadColumnTextWidth returns the estimated display width of the first column
func calculateHeadColumnTextWidth(widgets []Widget, columns int) int {
	maxLen := 0
	for i := range widgets {
		if label, ok := widgets[i].(Label); ok && i%columns == 0 {
			if textLen := calculateStringWidth(label.Text.(string)); textLen > maxLen {
				maxLen = textLen
			}
		}
	}
	return maxLen + 5
}

// calculateStringWidth returns the estimated display width of the given string
func calculateStringWidth(str string) int {
	return int(funk.Sum(funk.Map(util.RuneSizeInString(str), func(s int) int {
		// For better estimation, reduce size for non-ascii character
		if s > 1 {
			return s - 1
		}
		return s
	})) * 6)
}
