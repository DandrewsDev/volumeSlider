package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	"github.com/go-ole/go-ole"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/moutend/go-wca/pkg/wca"
	"github.com/spf13/viper"
	"golang.org/x/net/websocket"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"

)

var version = "latest"
var revision = "latest"


// Binding from JSON
type RunningExe struct {
	ID      int    `form:"id"      json:"id"     validate:""`
	Name    string `form:"name"    json:"name"   validate:"required"`
	Volume  int    `form:"volume"  json:"volume" validate:""`
}

type AppData struct {
	AppName    string `form:"appName"    json:"appName"   validate:"required"`
	ExeName    string `form:"exeName"    json:"exeName"   validate:"required"`
}

type ExeList struct {
	Exes []RunningExe `json:"processes"`
}

type AppList struct {
	AppNames []AppData `json:"appNames"`
}

var ExeListing ExeList

var e = echo.New()

func main() {

	go func() {
		desktopWindow()
		e.Shutdown(context.Background())
	}()

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return
	}

	var mmde *wca.IMMDeviceEnumerator
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return
	}
	defer mmde.Release()

	var mmd *wca.IMMDevice
	if err = mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		return
	}
	defer mmd.Release()

	var ps *wca.IPropertyStore
	if err = mmd.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err = ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return
	}

	var aev *wca.IAudioEndpointVolume
	if err = mmd.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return
	}
	defer aev.Release()

	e.HideBanner = true
	e.Use(middleware.Recover())

	e.GET("/app-vol/:exeName", func(c echo.Context) error {
		exeName := c.Param("exeName")
		appData := new(RunningExe)
		appData.Name = exeName
		if appData.Name == "master" {
			var masterVolumeLevelScalar float32
			if err = aev.GetMasterVolumeLevelScalar(&masterVolumeLevelScalar); err != nil {
				return c.JSON(http.StatusBadRequest, "failed to get master volume")
			}
			appData.Volume = int(masterVolumeLevelScalar * 100)
			return c.JSON(http.StatusOK, appData)
		}
		process, err := NewProcess(exeName)

		if err != nil {
			return c.String(http.StatusBadRequest, "failed to load process")
		}
		control, err := NewVolumeControl(process)
		if err != nil {
			return c.String(http.StatusBadRequest, "failed to control audio")
		}

		appData.Volume = getVolume(control)

		return c.JSON(http.StatusOK, appData)
	})
	e.POST("/app-vol-set", func(c echo.Context) error {
		rExe := new(RunningExe)

		if err = c.Bind(rExe); err != nil {
			return c.JSON(http.StatusBadRequest, "Failed to bind")
		}
		if len(rExe.Name) < 0 {
			return c.JSON(http.StatusBadRequest, rExe)
		}
		if rExe.Volume < 0 || rExe.Volume > 100 {
			return c.JSON(http.StatusBadRequest, "Failed to validate request")
		}

		if rExe.Name == "master" {
			if err = aev.SetMasterVolumeLevelScalar(float32(rExe.Volume)/100, nil); err != nil {
				return c.JSON(http.StatusBadRequest, err)
			}
			return c.JSON(http.StatusOK, rExe)
		}
		process, err := NewProcess(rExe.Name)

		if err != nil {
			return c.String(http.StatusBadRequest, rExe.Name)
		}
		control, err := NewVolumeControl(process)
		if err != nil {
			return c.String(http.StatusBadRequest, rExe.Name)
		}
		if err := control.SetVolume(rExe.Volume); err != nil {
			return c.String(http.StatusBadRequest, rExe.Name)
		}
		rExe.Volume = getVolume(control)
		return c.JSON(http.StatusOK, rExe)
	})

	e.GET("/slider-list", func(c echo.Context) error {
		return c.JSON(http.StatusOK, viper.Get("whitelist"))
	})
	e.POST("/slider-list", func(c echo.Context) error {
		list := new(AppList)
		if err = c.Bind(list); err != nil {
			return c.JSON(http.StatusBadRequest, "Failed to bind")
		}
		viper.Set("whitelist", list.AppNames)
		_ = viper.WriteConfig()
		return c.JSON(http.StatusOK, viper.Get("whitelist"))
	})
	e.HidePort = true
	defer ole.CoUninitialize()

	e.Use(middleware.CORS())
	// Start server
	e.Start(":1323")

}

// Vars injected via ldflags by bundler
var (
	VersionAstilectron string
	VersionElectron    string
)

type Process struct {
	pids []uint32
}

func graceShutdown() {
	_ = e.Shutdown(context.Background())
	os.Exit(1)
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// NewProcess finds a process from its name
func NewProcess(name string) (*Process, error) {
	pids := make([]uint32, 0)

	// Create a snapshot of all processes - TH32CS_SNAPPROCESS (0x00000002)
	handle, err := syscall.CreateToolhelp32Snapshot(0x00000002, 0)

	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := syscall.Process32First(handle, &entry); err != nil {
		return nil, err
	}

	if getProcessName(&entry) == name {
		pids = append(pids, entry.ProcessID)
	}

	for {
		if err := syscall.Process32Next(handle, &entry); err != nil {
			if err == syscall.ERROR_NO_MORE_FILES {
				break
			}

			return nil, err
		}

		if getProcessName(&entry) == name {
			pids = append(pids, entry.ProcessID)
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process with name '%s' found", name)
	}

	return &Process{
		pids: pids,
	}, nil
}

func (p *Process) hasPid(pid uint32) bool {
	for _, p := range p.pids {
		if p == pid {
			return true
		}
	}

	return false
}

func getProcessName(entry *syscall.ProcessEntry32) string {
	size := len(entry.ExeFile)

	for i := 0; i < size; i++ {
		if entry.ExeFile[i] == 0 {
			return syscall.UTF16ToString(entry.ExeFile[:i])
		}
	}

	return ""
}

func getProcessVolumes(process *Process) ([]*wca.ISimpleAudioVolume, error) {
	device, err := getAudioDevice()

	if err != nil {
		return nil, err
	}
	defer device.Release()

	enumerator, err := getSessionEnumerator(device)

	if err != nil {
		return nil, err
	}
	defer enumerator.Release()

	return getSessionVolumes(enumerator, process)
}

func getAudioDevice() (*wca.IMMDevice, error) {
	var deviceEnumerator *wca.IMMDeviceEnumerator

	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&deviceEnumerator,
	); err != nil {
		return nil, err
	}
	defer deviceEnumerator.Release()

	var device *wca.IMMDevice

	return device, deviceEnumerator.GetDefaultAudioEndpoint(
		wca.ERender,
		wca.EMultimedia,
		&device,
	)
}

func getSessionEnumerator(device *wca.IMMDevice) (*wca.IAudioSessionEnumerator, error) {
	var manager *wca.IAudioSessionManager2

	if err := device.Activate(
		wca.IID_IAudioSessionManager2,
		wca.CLSCTX_ALL,
		nil,
		&manager,
	); err != nil {
		return nil, err
	}
	defer manager.Release()

	var enumerator *wca.IAudioSessionEnumerator

	return enumerator, manager.GetSessionEnumerator(&enumerator)
}

func getSessionVolumes(enumerator *wca.IAudioSessionEnumerator, process *Process) ([]*wca.ISimpleAudioVolume, error) {
	var count int

	if err := enumerator.GetCount(&count); err != nil {
		return nil, err
	}

	volumes := make([]*wca.ISimpleAudioVolume, 0)

	for i := 0; i < count; i++ {
		var session *wca.IAudioSessionControl

		if err := enumerator.GetSession(i, &session); err != nil {
			return nil, err
		}

		dispatch, err := session.QueryInterface(wca.IID_IAudioSessionControl2)
		session.Release()

		if err != nil {
			return nil, err
		}

		session2 := (*wca.IAudioSessionControl2)(unsafe.Pointer(dispatch))

		var pid uint32

		if err := session2.GetProcessId(&pid); err != nil {
			// Ignore AUDCLNT_S_NO_CURRENT_PROCESS (0x889000D) - no error
			if err.(*ole.OleError).Code() != 0x889000D {
				session2.Release()

				continue
			}
		}

		if process.hasPid(pid) {
			dispatch, err := session2.QueryInterface(wca.IID_ISimpleAudioVolume)

			if err != nil {
				return nil, err
			}

			volume := (*wca.ISimpleAudioVolume)(unsafe.Pointer(dispatch))
			volumes = append(volumes, volume)
		}

		session2.Release()
	}

	return volumes, nil
}
// VolumeControl is used to control the volume of an individual process
type VolumeControl struct {
	volumes []*wca.ISimpleAudioVolume
}

// NewVolumeControl constructs a new control instance from a process
func NewVolumeControl(process *Process) (*VolumeControl, error) {
	volumes, err := getProcessVolumes(process)

	if err != nil {
		return nil, err
	}

	if len(volumes) == 0 {
		return nil, errors.New("no audio sessions found")
	}

	return &VolumeControl{
		volumes: volumes,
	}, nil
}

// GetVolume gets the volume level of the process
// Volume is in the range of 0 to 100
func (c *VolumeControl) GetVolume() (int, error) {
	var level float32

	if err := c.volumes[0].GetMasterVolume(&level); err != nil {
		return 0, err
	}

	return int(level * 100), nil
}

// SetVolume sets the volume level of the process
// Volume is in the range of 0 to 100
func (c *VolumeControl) SetVolume(level int) error {
	actualLevel := float32(level) / 100

	for _, volume := range c.volumes {
		if err := volume.SetMasterVolume(actualLevel, nil); err != nil {
			return err
		}
	}

	return nil
}

// Release frees the allocated resources
func (c *VolumeControl) Release() {
	for _, volume := range c.volumes {
		volume.Release()
	}
}

func getVolume(control *VolumeControl) int {
	level, err := control.GetVolume()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting volume: %s.", err)

		return 0
	}

	return level
}

func socketListen(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			// Write
			err := websocket.Message.Send(ws, "Hello, Client!")
			if err != nil {
				c.Logger().Error(err)
			}

			// Read
			msg := ""
			err = websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
			}
			fmt.Printf("%s\n", msg)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println("IP Issue")
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}


// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	switch m.Name {
	case "explore":
		// Unmarshal payload
		var path string
		if len(m.Payload) > 0 {
			// Unmarshal payload
			if err = json.Unmarshal(m.Payload, &path); err != nil {
				payload = err.Error()
				return
			}
		}
	}
	return
}

var (
	fs    = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	w     *astilectron.Window
	astil *astilectron.Astilectron
)

func desktopWindow() {
	// Create logger
	l := log.New(log.Writer(), log.Prefix(), log.Flags())

	// Parse flags
	fs.Parse(os.Args[1:])

	err := bootstrap.Run(bootstrap.Options{
		Asset:    Asset,
		AssetDir: AssetDir,
		AstilectronOptions: astilectron.Options{
			AppName:            "volumeSlider",
			AppIconDarwinPath:  "resources/app-icon.icns",
			AppIconDefaultPath: "resources/app-icon.png",
			SingleInstance:     true,
			VersionAstilectron: VersionAstilectron,
			VersionElectron:    VersionElectron,
		},
		Logger: l,
		TrayOptions: &astilectron.TrayOptions{
			Image:   astikit.StrPtr("resources/app-icon.png"),
			Tooltip: astikit.StrPtr("Tray's tooltip"),
		},
		TrayMenuOptions: []*astilectron.MenuItemOptions{
			{Label: astikit.StrPtr("Open Web Interface"), OnClick: func(e astilectron.Event) (deleteListener bool) {
				open("http://"+ GetOutboundIP() + ":1323")
				return
			}},
			{Label: astikit.StrPtr("Show UI Window"), OnClick: func(e astilectron.Event) (deleteListener bool) {
				w.Show()
				return
			}},
			{Label: astikit.StrPtr("Exit"), OnClick: func(e astilectron.Event) (deleteListener bool) {
				w.Destroy()
				graceShutdown()
				return
			}},
		},
		OnWait: func(ast *astilectron.Astilectron, ws []*astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			astil = ast
			w = ws[0]
			viper.SetConfigName("config")
			viper.SetConfigType("json")
			viper.AddConfigPath(ast.Paths().DataDirectory() + "/resources/")
			err := viper.ReadInConfig()
			if err != nil { // Handle errors reading the config file
				fmt.Println(err)
			}

			e.Static("/", ast.Paths().DataDirectory() + "/resources/app" )
			return nil
		},
		RestoreAssets: RestoreAssets,
		Windows: []*bootstrap.Window{{
			Homepage:       "index.html",
			MessageHandler: handleMessages,
			Options: &astilectron.WindowOptions{
				BackgroundColor: astikit.StrPtr("#333"),
				Center:          astikit.BoolPtr(true),
				Height:          astikit.IntPtr(550),
				Width:           astikit.IntPtr(800),
				Show:            flag.Bool("show", false, ""),
			},
		}},
	})


	if err != nil {
		l.Fatal(fmt.Errorf("running bootstrap failed: %w", err))
	}

}